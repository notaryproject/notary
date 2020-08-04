package rethinkdb

import (
	"errors"
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
)

var (
	errPoolClosed = errors.New("rethinkdb: pool is closed")
)

const (
	poolIsNotClosed int32 = 0
	poolIsClosed    int32 = 1
)

type connFactory func(host string, opts *ConnectOpts) (*Connection, error)

// A Pool is used to store a pool of connections to a single RethinkDB server
type Pool struct {
	host Host
	opts *ConnectOpts

	conns   []*Connection
	pointer int32
	closed  int32

	connFactory connFactory

	mu sync.Mutex // protects lazy creating connections
}

// NewPool creates a new connection pool for the given host
func NewPool(host Host, opts *ConnectOpts) (*Pool, error) {
	return newPool(host, opts, NewConnection)
}

func newPool(host Host, opts *ConnectOpts, connFactory connFactory) (*Pool, error) {
	initialCap := opts.InitialCap
	if initialCap <= 0 {
		// Fallback to MaxIdle if InitialCap is zero, this should be removed
		// when MaxIdle is removed
		initialCap = opts.MaxIdle
	}

	maxOpen := opts.MaxOpen
	if maxOpen <= 0 {
		maxOpen = 1
	}

	conns := make([]*Connection, maxOpen)
	var err error
	for i := 0; i < opts.InitialCap; i++ {
		conns[i], err = connFactory(host.String(), opts)
		if err != nil {
			return nil, err
		}
	}

	return &Pool{
		conns:       conns,
		pointer:     -1,
		host:        host,
		opts:        opts,
		connFactory: connFactory,
		closed:      poolIsNotClosed,
	}, nil
}

// Ping verifies a connection to the database is still alive,
// establishing a connection if necessary.
func (p *Pool) Ping() error {
	_, err := p.conn()
	return err
}

// Close closes the database, releasing any open resources.
//
// It is rare to Close a Pool, as the Pool handle is meant to be
// long-lived and shared between many goroutines.
func (p *Pool) Close() error {
	if atomic.LoadInt32(&p.closed) == poolIsClosed {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed == poolIsClosed {
		return nil
	}
	p.closed = poolIsClosed

	for _, c := range p.conns {
		if c != nil {
			err := c.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Pool) conn() (*Connection, error) {
	if atomic.LoadInt32(&p.closed) == poolIsClosed {
		return nil, errPoolClosed
	}

	pos := atomic.AddInt32(&p.pointer, 1)
	if pos == int32(len(p.conns)) {
		atomic.StoreInt32(&p.pointer, 0)
	}
	pos = pos % int32(len(p.conns))

	var err error

	if p.conns[pos] == nil {
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.conns[pos] == nil {
			p.conns[pos], err = p.connFactory(p.host.String(), p.opts)
			if err != nil {
				return nil, err
			}
		}
	} else if p.conns[pos].isBad() {
		// connBad connection needs to be reconnected
		p.mu.Lock()
		defer p.mu.Unlock()

		p.conns[pos], err = p.connFactory(p.host.String(), p.opts)
		if err != nil {
			return nil, err
		}
	}

	return p.conns[pos], nil
}

// SetInitialPoolCap sets the initial capacity of the connection pool.
//
// Deprecated: This value should only be set when connecting
func (p *Pool) SetInitialPoolCap(n int) {
	return
}

// SetMaxIdleConns sets the maximum number of connections in the idle
// connection pool.
//
// Deprecated: This value should only be set when connecting
func (p *Pool) SetMaxIdleConns(n int) {
	return
}

// SetMaxOpenConns sets the maximum number of open connections to the database.
//
// Deprecated: This value should only be set when connecting
func (p *Pool) SetMaxOpenConns(n int) {
	return
}

// Query execution functions

// Exec executes a query without waiting for any response.
func (p *Pool) Exec(ctx context.Context, q Query) error {
	c, err := p.conn()
	if err != nil {
		return err
	}

	_, _, err = c.Query(ctx, q)
	return err
}

// Query executes a query and waits for the response
func (p *Pool) Query(ctx context.Context, q Query) (*Cursor, error) {
	c, err := p.conn()
	if err != nil {
		return nil, err
	}

	_, cursor, err := c.Query(ctx, q)
	return cursor, err
}

// Server returns the server name and server UUID being used by a connection.
func (p *Pool) Server() (ServerResponse, error) {
	var response ServerResponse

	c, err := p.conn()
	if err != nil {
		return response, err
	}

	response, err = c.Server()
	return response, err
}
