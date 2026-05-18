package ws

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var ErrNilConn = errors.New("websocket connection is nil")

type SafeConn struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	closed bool
}

func NewSafeConn(conn *websocket.Conn) *SafeConn {
	return &SafeConn{conn: conn}
}

func (c *SafeConn) WriteJSON(v interface{}) error {
	if c == nil {
		return ErrNilConn
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.closed {
		return ErrNilConn
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return c.conn.WriteJSON(v)
}

func (c *SafeConn) ReadJSON(v interface{}) error {
	if c == nil || c.conn == nil {
		return ErrNilConn
	}
	return c.conn.ReadJSON(v)
}

func (c *SafeConn) Close() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.closed {
		return nil
	}

	err := c.conn.Close()
	c.closed = true
	return err
}

func (c *SafeConn) SetReadLimit(limit int64) {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.SetReadLimit(limit)
}

func (c *SafeConn) SetReadDeadline(t time.Time) error {
	if c == nil || c.conn == nil {
		return ErrNilConn
	}
	return c.conn.SetReadDeadline(t)
}

func (c *SafeConn) SetPongHandler(handler func(string) error) {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.SetPongHandler(handler)
}
