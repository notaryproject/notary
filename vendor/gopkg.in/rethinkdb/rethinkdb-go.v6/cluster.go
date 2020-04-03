package rethinkdb

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hailocab/go-hostpool"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"gopkg.in/cenkalti/backoff.v2"
)

var errClusterClosed = errors.New("rethinkdb: cluster is closed")

const (
	clusterWorking = 0
	clusterClosed  = 1
)

// A Cluster represents a connection to a RethinkDB cluster, a cluster is created
// by the Session and should rarely be created manually.
//
// The cluster keeps track of all nodes in the cluster and if requested can listen
// for cluster changes and start tracking a new node if one appears. Currently
// nodes are removed from the pool if they become unhealthy (100 failed queries).
// This should hopefully soon be replaced by a backoff system.
type Cluster struct {
	opts *ConnectOpts

	mu     sync.RWMutex
	seeds  []Host // Initial host nodes specified by user.
	hp     hostpool.HostPool
	nodes  map[string]*Node // Active nodes in cluster.
	closed int32            // 0 - working, 1 - closed

	connFactory connFactory

	discoverInterval time.Duration
}

// NewCluster creates a new cluster by connecting to the given hosts.
func NewCluster(hosts []Host, opts *ConnectOpts) (*Cluster, error) {
	c := &Cluster{
		hp:          newHostPool(opts),
		seeds:       hosts,
		opts:        opts,
		closed:      clusterWorking,
		connFactory: NewConnection,
	}

	err := c.run()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func newHostPool(opts *ConnectOpts) hostpool.HostPool {
	return hostpool.NewEpsilonGreedy([]string{}, opts.HostDecayDuration, &hostpool.LinearEpsilonValueCalculator{})
}

func (c *Cluster) run() error {
	// Attempt to connect to each host and discover any additional hosts if host
	// discovery is enabled
	if err := c.connectCluster(); err != nil {
		return err
	}

	if !c.IsConnected() {
		return ErrNoConnectionsStarted
	}
	return nil
}

// Query executes a ReQL query using the cluster to connect to the database
func (c *Cluster) Query(ctx context.Context, q Query) (cursor *Cursor, err error) {
	for i := 0; i < c.numRetries(); i++ {
		var node *Node
		var hpr hostpool.HostPoolResponse

		node, hpr, err = c.GetNextNode()
		if err != nil {
			return nil, err
		}

		cursor, err = node.Query(ctx, q)
		hpr.Mark(err)

		if !shouldRetryQuery(q, err) {
			break
		}
	}

	return cursor, err
}

// Exec executes a ReQL query using the cluster to connect to the database
func (c *Cluster) Exec(ctx context.Context, q Query) (err error) {
	for i := 0; i < c.numRetries(); i++ {
		var node *Node
		var hpr hostpool.HostPoolResponse

		node, hpr, err = c.GetNextNode()
		if err != nil {
			return err
		}

		err = node.Exec(ctx, q)
		hpr.Mark(err)

		if !shouldRetryQuery(q, err) {
			break
		}
	}

	return err
}

// Server returns the server name and server UUID being used by a connection.
func (c *Cluster) Server() (response ServerResponse, err error) {
	for i := 0; i < c.numRetries(); i++ {
		var node *Node
		var hpr hostpool.HostPoolResponse

		node, hpr, err = c.GetNextNode()
		if err != nil {
			return ServerResponse{}, err
		}

		response, err = node.Server()
		hpr.Mark(err)

		// This query should not fail so retry if any error is detected
		if err == nil {
			break
		}
	}

	return response, err
}

// SetInitialPoolCap sets the initial capacity of the connection pool.
func (c *Cluster) SetInitialPoolCap(n int) {
	for _, node := range c.GetNodes() {
		node.SetInitialPoolCap(n)
	}
}

// SetMaxIdleConns sets the maximum number of connections in the idle
// connection pool.
func (c *Cluster) SetMaxIdleConns(n int) {
	for _, node := range c.GetNodes() {
		node.SetMaxIdleConns(n)
	}
}

// SetMaxOpenConns sets the maximum number of open connections to the database.
func (c *Cluster) SetMaxOpenConns(n int) {
	for _, node := range c.GetNodes() {
		node.SetMaxOpenConns(n)
	}
}

// Close closes the cluster
func (c *Cluster) Close(optArgs ...CloseOpts) error {
	if c.isClosed() {
		return nil
	}

	for _, node := range c.GetNodes() {
		err := node.Close(optArgs...)
		if err != nil {
			return err
		}
	}

	c.hp.Close()
	atomic.StoreInt32(&c.closed, clusterClosed)

	return nil
}

func (c *Cluster) isClosed() bool {
	return atomic.LoadInt32(&c.closed) == clusterClosed
}

// discover attempts to find new nodes in the cluster using the current nodes
func (c *Cluster) discover() {
	// Keep retrying with exponential backoff.
	b := backoff.NewExponentialBackOff()
	// Never finish retrying (max interval is still 60s)
	b.MaxElapsedTime = 0
	if c.discoverInterval != 0 {
		b.InitialInterval = c.discoverInterval
	}

	// Keep trying to discover new nodes
	for {
		if c.isClosed() {
			return
		}

		_ = backoff.RetryNotify(func() error {
			if c.isClosed() {
				return backoff.Permanent(errClusterClosed)
			}
			// If no hosts try seeding nodes
			if len(c.GetNodes()) == 0 {
				return c.connectCluster()
			}

			return c.listenForNodeChanges()
		}, b, func(err error, wait time.Duration) {
			Log.Debugf("Error discovering hosts %s, waiting: %s", err, wait)
		})
	}
}

// listenForNodeChanges listens for changes to node status using change feeds.
// This function will block until the query fails
func (c *Cluster) listenForNodeChanges() error {
	// Start listening to changes from a random active node
	node, hpr, err := c.GetNextNode()
	if err != nil {
		return err
	}

	q, err := newQuery(
		DB(SystemDatabase).Table(ServerStatusSystemTable).Changes(ChangesOpts{IncludeInitial: true}),
		map[string]interface{}{},
		c.opts,
	)
	if err != nil {
		return fmt.Errorf("Error building query: %s", err)
	}

	cursor, err := node.Query(context.Background(), q) // no need for timeout due to Changes()
	if err != nil {
		hpr.Mark(err)
		return err
	}
	defer func() { _ = cursor.Close() }()

	// Keep reading node status updates from changefeed
	var result struct {
		NewVal *nodeStatus `rethinkdb:"new_val"`
		OldVal *nodeStatus `rethinkdb:"old_val"`
	}
	for cursor.Next(&result) {
		addr := fmt.Sprintf("%s:%d", result.NewVal.Network.Hostname, result.NewVal.Network.ReqlPort)
		addr = strings.ToLower(addr)

		if result.NewVal != nil && result.OldVal == nil {
			// added new node
			if !c.nodeExists(result.NewVal.ID) {
				// Connect to node using exponential backoff (give up after waiting 5s)
				// to give the node time to start-up.
				b := backoff.NewExponentialBackOff()
				b.MaxElapsedTime = time.Second * 5

				err = backoff.Retry(func() error {
					node, err := c.connectNodeWithStatus(result.NewVal)
					if err == nil {
						c.addNode(node)

						Log.WithFields(logrus.Fields{
							"id":   node.ID,
							"host": node.Host.String(),
						}).Debug("Connected to node")
					}
					return err
				}, b)
				if err != nil {
					return err
				}
			}
		} else if result.OldVal != nil && result.NewVal == nil {
			// removed old node
			oldNode := c.removeNode(result.OldVal.ID)
			if oldNode != nil {
				_ = oldNode.Close()
			}
		} else {
			// node updated
			// nothing to do - assuming node can't change it's hostname in a single Changes() message
		}
	}

	err = cursor.Err()
	hpr.Mark(err)
	return err
}

func (c *Cluster) connectCluster() error {
	nodeSet := map[string]*Node{}
	var attemptErr error

	// Attempt to connect to each seed host
	for _, host := range c.seeds {
		conn, err := c.connFactory(host.String(), c.opts)
		if err != nil {
			attemptErr = err
			Log.Warnf("Error creating connection: %s", err.Error())
			continue
		}

		svrRsp, err := conn.Server()
		if err != nil {
			attemptErr = err
			Log.Warnf("Error fetching server ID: %s", err)
			_ = conn.Close()

			continue
		}
		_ = conn.Close()

		node, err := c.connectNode(svrRsp.ID, []Host{host})
		if err != nil {
			attemptErr = err
			Log.Warnf("Error connecting to node: %s", err)
			continue
		}

		if _, ok := nodeSet[node.ID]; !ok {
			Log.WithFields(logrus.Fields{
				"id":   node.ID,
				"host": node.Host.String(),
			}).Debug("Connected to node")

			nodeSet[node.ID] = node
		} else {
			// dublicate node
			_ = node.Close()
		}
	}

	// If no nodes were contactable then return the last error, this does not
	// include driver errors such as if there was an issue building the
	// query
	if len(nodeSet) == 0 {
		if attemptErr != nil {
			return attemptErr
		}
		return ErrNoConnections
	}

	var nodes []*Node
	for _, node := range nodeSet {
		nodes = append(nodes, node)
	}
	c.replaceNodes(nodes)

	if c.opts.DiscoverHosts {
		go c.discover()
	}

	return nil
}

func (c *Cluster) connectNodeWithStatus(s *nodeStatus) (*Node, error) {
	aliases := make([]Host, len(s.Network.CanonicalAddresses))
	for i, aliasAddress := range s.Network.CanonicalAddresses {
		aliases[i] = NewHost(aliasAddress.Host, int(s.Network.ReqlPort))
	}

	return c.connectNode(s.ID, aliases)
}

func (c *Cluster) connectNode(id string, aliases []Host) (*Node, error) {
	var pool *Pool
	var err error

	for len(aliases) > 0 {
		pool, err = newPool(aliases[0], c.opts, c.connFactory)
		if err != nil {
			aliases = aliases[1:]
			continue
		}

		err = pool.Ping()
		if err != nil {
			aliases = aliases[1:]
			continue
		}

		// Ping successful so break out of loop
		break
	}

	if err != nil {
		return nil, err
	}
	if len(aliases) == 0 {
		return nil, ErrInvalidNode
	}

	return newNode(id, aliases, pool), nil
}

// IsConnected returns true if cluster has nodes and is not already connClosed.
func (c *Cluster) IsConnected() bool {
	return (len(c.GetNodes()) > 0) && !c.isClosed()
}

// GetNextNode returns a random node on the cluster
func (c *Cluster) GetNextNode() (*Node, hostpool.HostPoolResponse, error) {
	if !c.IsConnected() {
		return nil, nil, ErrNoConnections
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := c.nodes
	hpr := c.hp.Get()
	if n, ok := nodes[hpr.Host()]; ok {
		if !n.Closed() {
			return n, hpr, nil
		}
	}

	return nil, nil, ErrNoConnections
}

// GetNodes returns a list of all nodes in the cluster
func (c *Cluster) GetNodes() []*Node {
	c.mu.RLock()
	defer c.mu.RUnlock()
	nodes := make([]*Node, 0, len(c.nodes))
	for _, n := range c.nodes {
		nodes = append(nodes, n)
	}

	return nodes
}

func (c *Cluster) nodeExists(nodeID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, node := range c.nodes {
		if node.ID == nodeID {
			return true
		}
	}
	return false
}

func (c *Cluster) addNode(node *Node) {
	host := node.Host.String()
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exist := c.nodes[host]; exist {
		// addNode() should be called only if the node doesn't exist
		return
	}

	c.nodes[host] = node

	hosts := make([]string, 0, len(c.nodes))
	for _, n := range c.nodes {
		hosts = append(hosts, n.Host.String())
	}
	c.hp.SetHosts(hosts)
}

func (c *Cluster) replaceNodes(nodes []*Node) {
	nodesMap := make(map[string]*Node, len(nodes))
	hosts := make([]string, len(nodes))
	for i, node := range nodes {
		host := node.Host.String()

		nodesMap[host] = node
		hosts[i] = host
	}

	sort.Strings(hosts) // unit tests stability

	c.mu.Lock()
	c.nodes = nodesMap
	c.hp.SetHosts(hosts)
	c.mu.Unlock()
}

func (c *Cluster) removeNode(nodeID string) *Node {
	c.mu.Lock()
	defer c.mu.Unlock()
	var rmNode *Node
	for _, node := range c.nodes {
		if node.ID == nodeID {
			rmNode = node
			break
		}
	}
	if rmNode == nil {
		return nil
	}

	delete(c.nodes, rmNode.Host.String())

	hosts := make([]string, 0, len(c.nodes))
	for _, n := range c.nodes {
		hosts = append(hosts, n.Host.String())
	}
	c.hp.SetHosts(hosts)

	return rmNode
}

func (c *Cluster) numRetries() int {
	if n := c.opts.NumRetries; n > 0 {
		return n
	}

	return 3
}
