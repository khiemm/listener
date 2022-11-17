package common

import (
	"io"
	"net"
	"time"
)

// TeeConn returns a net.Conn that writes to dst what it reads from conn.
// Internally, it uses io.TeeReader so you can look at that if you need
// more information.
func TeeConn(conn net.Conn, dst io.Writer) net.Conn {
	return teeConn{conn, io.TeeReader(conn, dst)}
}

type teeConn struct {
	conn   net.Conn
	reader io.Reader
}

func (tc teeConn) Read(b []byte) (n int, err error)   { return tc.reader.Read(b) }
func (tc teeConn) Write(b []byte) (n int, err error)  { return tc.conn.Write(b) }
func (tc teeConn) Close() error                       { return tc.conn.Close() }
func (tc teeConn) LocalAddr() net.Addr                { return tc.conn.LocalAddr() }
func (tc teeConn) RemoteAddr() net.Addr               { return tc.conn.RemoteAddr() }
func (tc teeConn) SetDeadline(t time.Time) error      { return tc.conn.SetDeadline(t) }
func (tc teeConn) SetReadDeadline(t time.Time) error  { return tc.conn.SetReadDeadline(t) }
func (tc teeConn) SetWriteDeadline(t time.Time) error { return tc.conn.SetWriteDeadline(t) }
