package rethinkdb

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"bytes"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"golang.org/x/net/context"
	p "gopkg.in/rethinkdb/rethinkdb-go.v6/ql2"
	"sync"
)

const (
	respHeaderLen          = 12
	defaultKeepAlivePeriod = time.Second * 30

	connNotBad = 0
	connBad    = 1

	connWorking = 0
	connClosed  = 1
)

// Response represents the raw response from a query, most of the time you
// should instead use a Cursor when reading from the database.
type Response struct {
	Token     int64
	Type      p.Response_ResponseType   `json:"t"`
	ErrorType p.Response_ErrorType      `json:"e"`
	Notes     []p.Response_ResponseNote `json:"n"`
	Responses []json.RawMessage         `json:"r"`
	Backtrace []interface{}             `json:"b"`
	Profile   interface{}               `json:"p"`
}

// Connection is a connection to a rethinkdb database. Connection is not thread
// safe and should only be accessed be a single goroutine
type Connection struct {
	net.Conn

	address string
	opts    *ConnectOpts

	_                  [4]byte
	token              int64
	cursors            map[int64]*Cursor
	bad                int32 // 0 - not bad, 1 - bad
	closed             int32 // 0 - working, 1 - closed
	stopReadChan       chan bool
	readRequestsChan   chan tokenAndPromise
	responseChan       chan responseAndError
	stopProcessingChan chan struct{}
	mu                 sync.Mutex
}

type responseAndError struct {
	response *Response
	err      error
}

type responseAndCursor struct {
	response *Response
	cursor   *Cursor
	err      error
}

type tokenAndPromise struct {
	ctx     context.Context
	query   *Query
	promise chan responseAndCursor
	span    opentracing.Span
}

// NewConnection creates a new connection to the database server
func NewConnection(address string, opts *ConnectOpts) (*Connection, error) {
	keepAlivePeriod := defaultKeepAlivePeriod
	if opts.KeepAlivePeriod > 0 {
		keepAlivePeriod = opts.KeepAlivePeriod
	}

	// Connect to Server
	var err error
	var conn net.Conn
	nd := net.Dialer{Timeout: opts.Timeout, KeepAlive: keepAlivePeriod}
	if opts.TLSConfig == nil {
		conn, err = nd.Dial("tcp", address)
	} else {
		conn, err = tls.DialWithDialer(&nd, "tcp", address, opts.TLSConfig)
	}
	if err != nil {
		return nil, RQLConnectionError{rqlError(err.Error())}
	}

	c := newConnection(conn, address, opts)

	// Send handshake
	handshake, err := c.handshake(opts.HandshakeVersion)
	if err != nil {
		return nil, err
	}

	if err = handshake.Send(); err != nil {
		return nil, err
	}

	// NOTE: mock.go: Mock.Query()
	// NOTE: connection_test.go: runConnection()
	go c.readSocket()
	go c.processResponses()

	return c, nil
}

func newConnection(conn net.Conn, address string, opts *ConnectOpts) *Connection {
	c := &Connection{
		Conn:               conn,
		address:            address,
		opts:               opts,
		cursors:            make(map[int64]*Cursor),
		stopReadChan:       make(chan bool, 1),
		bad:                connNotBad,
		closed:             connWorking,
		readRequestsChan:   make(chan tokenAndPromise, 16),
		responseChan:       make(chan responseAndError, 16),
		stopProcessingChan: make(chan struct{}),
	}
	return c
}

// Close closes the underlying net.Conn
func (c *Connection) Close() error {
	var err error

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isClosed() {
		c.setClosed()
		close(c.stopReadChan)
		err = c.Conn.Close()
	}

	return err
}

// Query sends a Query to the database, returning both the raw Response and a
// Cursor which should be used to view the query's response.
//
// This function is used internally by Run which should be used for most queries.
func (c *Connection) Query(ctx context.Context, q Query) (*Response, *Cursor, error) {
	if c == nil {
		return nil, nil, ErrConnectionClosed
	}
	if c.Conn == nil || c.isClosed() {
		c.setBad()
		return nil, nil, ErrConnectionClosed
	}
	if ctx == nil {
		ctx = c.contextFromConnectionOpts()
	}

	// Add token if query is a START/NOREPLY_WAIT
	if q.Type == p.Query_START || q.Type == p.Query_NOREPLY_WAIT || q.Type == p.Query_SERVER_INFO {
		q.Token = c.nextToken()
	}
	if q.Type == p.Query_START || q.Type == p.Query_NOREPLY_WAIT {
		if c.opts.Database != "" {
			var err error
			q.Opts["db"], err = DB(c.opts.Database).Build()
			if err != nil {
				return nil, nil, RQLDriverError{rqlError(err.Error())}
			}
		}
	}

	var fetchingSpan opentracing.Span
	if c.opts.UseOpentracing {
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			if q.Type == p.Query_START {
				querySpan := c.startTracingSpan(parentSpan, &q) // will be Finished when cursor connClosed
				parentSpan = querySpan
				ctx = opentracing.ContextWithSpan(ctx, querySpan)
			}

			fetchingSpan = c.startTracingSpan(parentSpan, &q) // will be Finished when response arrived
		}
	}

	err := c.sendQuery(q)
	if err != nil {
		if fetchingSpan != nil {
			ext.Error.Set(fetchingSpan, true)
			fetchingSpan.LogFields(log.Error(err))
			fetchingSpan.Finish()
			if q.Type == p.Query_START {
				opentracing.SpanFromContext(ctx).Finish()
			}
		}
		return nil, nil, err
	}

	if noreply, ok := q.Opts["noreply"]; ok && noreply.(bool) {
		return nil, nil, nil
	}

	promise := make(chan responseAndCursor, 1)
	select {
	case c.readRequestsChan <- tokenAndPromise{ctx: ctx, query: &q, span: fetchingSpan, promise: promise}:
	case <-ctx.Done():
		return c.stopQuery(&q)
	}

	select {
	case future := <-promise:
		return future.response, future.cursor, future.err
	case <-ctx.Done():
		return c.stopQuery(&q)
	case <-c.stopProcessingChan: // connection readRequests processing stopped, promise can be never answered
		return nil, nil, ErrConnectionClosed
	}
}

func (c *Connection) stopQuery(q *Query) (*Response, *Cursor, error) {
	if q.Type != p.Query_STOP && !c.isClosed() && !c.isBad() {
		stopQuery := newStopQuery(q.Token)
		_, _, _ = c.Query(c.contextFromConnectionOpts(), stopQuery)
	}
	return nil, nil, ErrQueryTimeout
}

func (c *Connection) startTracingSpan(parentSpan opentracing.Span, q *Query) opentracing.Span {
	span := parentSpan.Tracer().StartSpan(
		"Query_"+q.Type.String(),
		opentracing.ChildOf(parentSpan.Context()),
		ext.SpanKindRPCClient)

	ext.PeerAddress.Set(span, c.address)
	ext.Component.Set(span, "rethinkdb-go")

	if q.Type == p.Query_START {
		span.LogFields(log.String("query", q.Term.String()))
	}

	return span
}

func (c *Connection) readSocket() {
	for {
		response, err := c.readResponse()

		c.responseChan <- responseAndError{
			response: response,
			err:      err,
		}

		select {
		case <-c.stopReadChan:
			close(c.responseChan)
			return
		default:
		}
	}
}

func (c *Connection) processResponses() {
	readRequests := make([]tokenAndPromise, 0, 16)
	responses := make([]*Response, 0, 16)
	for {
		var response *Response
		var readRequest tokenAndPromise
		var ok bool

		select {
		case respPair, openned := <-c.responseChan:
			if respPair.err != nil {
				// Transport socket error, can't continue to work
				// Don't know return to who (no token) - return to all
				broadcastError(readRequests, respPair.err)
				readRequests = []tokenAndPromise{}
				_ = c.Close() // next `if` will be called indirect cascade by closing chans
				continue
			}
			if !openned { // responseChan is connClosed (stopReadChan is closed too)
				close(c.stopProcessingChan)
				broadcastError(readRequests, ErrConnectionClosed)
				c.cursors = nil

				return
			}

			response = respPair.response

			readRequest, ok = getReadRequest(readRequests, respPair.response.Token)
			if !ok {
				responses = append(responses, respPair.response)
				continue
			}
			readRequests = removeReadRequest(readRequests, respPair.response.Token)

		case readRequest = <-c.readRequestsChan:
			response, ok = getResponse(responses, readRequest.query.Token)
			if !ok {
				readRequests = append(readRequests, readRequest)
				continue
			}
			responses = removeResponse(responses, readRequest.query.Token)
		}

		response, cursor, err := c.processResponse(readRequest.ctx, *readRequest.query, response, readRequest.span)
		if readRequest.promise != nil {
			readRequest.promise <- responseAndCursor{response: response, cursor: cursor, err: err}
			close(readRequest.promise)
		}
	}
}

func broadcastError(readRequests []tokenAndPromise, err error) {
	for _, rr := range readRequests {
		if rr.promise != nil {
			rr.promise <- responseAndCursor{err: err}
			close(rr.promise)
		}
	}
}

type ServerResponse struct {
	ID   string `rethinkdb:"id"`
	Name string `rethinkdb:"name"`
}

// Server returns the server name and server UUID being used by a connection.
func (c *Connection) Server() (ServerResponse, error) {
	var response ServerResponse

	_, cur, err := c.Query(c.contextFromConnectionOpts(), Query{
		Type: p.Query_SERVER_INFO,
	})
	if err != nil {
		return response, err
	}

	if err = cur.One(&response); err != nil {
		return response, err
	}

	if err = cur.Close(); err != nil {
		return response, err
	}

	return response, nil
}

// sendQuery marshals the Query and sends the JSON to the server.
func (c *Connection) sendQuery(q Query) error {
	buf := &bytes.Buffer{}
	buf.Grow(respHeaderLen)
	buf.Write(buf.Bytes()[:respHeaderLen]) // reserve for header
	enc := json.NewEncoder(buf)

	// Build query
	err := enc.Encode(q.Build())
	if err != nil {
		return RQLDriverError{rqlError(fmt.Sprintf("Error building query: %s", err.Error()))}
	}

	b := buf.Bytes()

	// Write header
	binary.LittleEndian.PutUint64(b, uint64(q.Token))
	binary.LittleEndian.PutUint32(b[8:], uint32(len(b)-respHeaderLen))

	// Send the JSON encoding of the query itself.
	if err = c.writeData(b); err != nil {
		c.setBad()
		return RQLConnectionError{rqlError(err.Error())}
	}

	return nil
}

// getToken generates the next query token, used to number requests and match
// responses with requests.
func (c *Connection) nextToken() int64 {
	// requires c.token to be 64-bit aligned on ARM
	return atomic.AddInt64(&c.token, 1)
}

// readResponse attempts to read a Response from the server, if no response
// could be read then an error is returned.
func (c *Connection) readResponse() (*Response, error) {
	// due to this is pooled connection, it always reads from socket even if idle
	// timeouts should be only on query-level with context

	// Read response header (token+length)
	headerBuf := [respHeaderLen]byte{}
	if _, err := c.read(headerBuf[:]); err != nil {
		c.setBad()
		return nil, RQLConnectionError{rqlError(err.Error())}
	}

	responseToken := int64(binary.LittleEndian.Uint64(headerBuf[:8]))
	messageLength := binary.LittleEndian.Uint32(headerBuf[8:])

	// Read the JSON encoding of the Response itself.
	b := make([]byte, int(messageLength))

	if _, err := c.read(b); err != nil {
		c.setBad()
		return nil, RQLConnectionError{rqlError(err.Error())}
	}

	// Decode the response
	var response = new(Response)
	if err := json.Unmarshal(b, response); err != nil {
		c.setBad()
		return nil, RQLDriverError{rqlError(err.Error())}
	}
	response.Token = responseToken

	return response, nil
}

// Called to fill response for the query
func (c *Connection) processResponse(ctx context.Context, q Query, response *Response, span opentracing.Span) (r *Response, cur *Cursor, err error) {
	if span != nil {
		defer func() {
			if err != nil {
				ext.Error.Set(span, true)
				span.LogFields(log.Error(err))
			}
			span.Finish()
		}()
	}

	switch response.Type {
	case p.Response_CLIENT_ERROR:
		return response, c.processErrorResponse(response), createClientError(response, q.Term)
	case p.Response_COMPILE_ERROR:
		return response, c.processErrorResponse(response), createCompileError(response, q.Term)
	case p.Response_RUNTIME_ERROR:
		return response, c.processErrorResponse(response), createRuntimeError(response.ErrorType, response, q.Term)
	case p.Response_SUCCESS_ATOM, p.Response_SERVER_INFO:
		return c.processAtomResponse(ctx, q, response)
	case p.Response_SUCCESS_PARTIAL:
		return c.processPartialResponse(ctx, q, response)
	case p.Response_SUCCESS_SEQUENCE:
		return c.processSequenceResponse(ctx, q, response)
	case p.Response_WAIT_COMPLETE:
		return c.processWaitResponse(response)
	default:
		return nil, nil, RQLDriverError{rqlError(fmt.Sprintf("Unexpected response type: %v", response.Type.String()))}
	}
}

func (c *Connection) processErrorResponse(response *Response) *Cursor {
	cursor := c.cursors[response.Token]
	delete(c.cursors, response.Token)
	return cursor
}

func (c *Connection) processAtomResponse(ctx context.Context, q Query, response *Response) (*Response, *Cursor, error) {
	cursor := newCursor(ctx, c, "Cursor", response.Token, q.Term, q.Opts)
	cursor.profile = response.Profile
	cursor.extend(response)

	return response, cursor, nil
}

func (c *Connection) processPartialResponse(ctx context.Context, q Query, response *Response) (*Response, *Cursor, error) {
	cursorType := "Cursor"
	if len(response.Notes) > 0 {
		switch response.Notes[0] {
		case p.Response_SEQUENCE_FEED:
			cursorType = "Feed"
		case p.Response_ATOM_FEED:
			cursorType = "AtomFeed"
		case p.Response_ORDER_BY_LIMIT_FEED:
			cursorType = "OrderByLimitFeed"
		case p.Response_UNIONED_FEED:
			cursorType = "UnionedFeed"
		case p.Response_INCLUDES_STATES:
			cursorType = "IncludesFeed"
		}
	}

	cursor, ok := c.cursors[response.Token]
	if !ok {
		// Create a new cursor if needed
		cursor = newCursor(ctx, c, cursorType, response.Token, q.Term, q.Opts)
		cursor.profile = response.Profile

		c.cursors[response.Token] = cursor
	}

	cursor.extend(response)

	return response, cursor, nil
}

func (c *Connection) processSequenceResponse(ctx context.Context, q Query, response *Response) (*Response, *Cursor, error) {
	cursor, ok := c.cursors[response.Token]
	if !ok {
		// Create a new cursor if needed
		cursor = newCursor(ctx, c, "Cursor", response.Token, q.Term, q.Opts)
		cursor.profile = response.Profile
	}
	delete(c.cursors, response.Token)

	cursor.extend(response)

	return response, cursor, nil
}

func (c *Connection) processWaitResponse(response *Response) (*Response, *Cursor, error) {
	delete(c.cursors, response.Token)
	return response, nil, nil
}

func (c *Connection) setBad() {
	atomic.StoreInt32(&c.bad, connBad)
}

func (c *Connection) isBad() bool {
	return atomic.LoadInt32(&c.bad) == connBad
}

func (c *Connection) setClosed() {
	atomic.StoreInt32(&c.closed, connClosed)
}

func (c *Connection) isClosed() bool {
	return atomic.LoadInt32(&c.closed) == connClosed
}

func getReadRequest(list []tokenAndPromise, token int64) (tokenAndPromise, bool) {
	for _, e := range list {
		if e.query.Token == token {
			return e, true
		}
	}
	return tokenAndPromise{}, false
}

func getResponse(list []*Response, token int64) (*Response, bool) {
	for _, e := range list {
		if e.Token == token {
			return e, true
		}
	}
	return nil, false
}

func removeReadRequest(list []tokenAndPromise, token int64) []tokenAndPromise {
	for i := range list {
		if list[i].query.Token == token {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func removeResponse(list []*Response, token int64) []*Response {
	for i := range list {
		if list[i].Token == token {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}
