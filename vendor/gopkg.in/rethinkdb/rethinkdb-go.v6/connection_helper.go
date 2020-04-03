package rethinkdb

import (
	"golang.org/x/net/context"
	"io"
)

// Write 'data' to conn
func (c *Connection) writeData(data []byte) error {
	_, err := c.Conn.Write(data[:])

	return err
}

func (c *Connection) read(buf []byte) (total int, err error) {
	return io.ReadFull(c.Conn, buf)
}

func (c *Connection) contextFromConnectionOpts() context.Context {
	// back compatibility
	min := c.opts.ReadTimeout
	if c.opts.WriteTimeout < min {
		min = c.opts.WriteTimeout
	}
	if min == 0 {
		return context.Background()
	}
	ctx, _ := context.WithTimeout(context.Background(), min)
	return ctx
}
