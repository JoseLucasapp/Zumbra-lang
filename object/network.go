package object

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// NetListener is a TCP listener with an optional TLS server configuration.
// Deadlines are applied to the underlying TCP listener so accept timeouts work
// identically for plain TCP and TLS.
type NetListener struct {
	Listener  *net.TCPListener
	TLSConfig *tls.Config
	closed    atomic.Bool
}

func (l *NetListener) Type() ObjectType { return NET_LISTENER_OBJ }
func (l *NetListener) Inspect() string {
	if l == nil || l.Listener == nil {
		return "NetListener<nil>"
	}
	protocol := "tcp"
	if l.TLSConfig != nil {
		protocol = "tls"
	}
	return fmt.Sprintf("NetListener<%s %s>", protocol, l.Listener.Addr().String())
}
func (l *NetListener) Close() error {
	if l == nil || l.Listener == nil || !l.closed.CompareAndSwap(false, true) {
		return nil
	}
	return l.Listener.Close()
}
func (l *NetListener) Closed() bool { return l == nil || l.closed.Load() }
func (l *NetListener) Accept(timeout time.Duration) (*NetStream, bool, error) {
	if l == nil || l.Listener == nil {
		return nil, false, fmt.Errorf("listener is nil")
	}
	if timeout >= 0 {
		if err := l.Listener.SetDeadline(time.Now().Add(timeout)); err != nil {
			return nil, false, err
		}
		defer l.Listener.SetDeadline(time.Time{})
	}
	conn, err := l.Listener.AcceptTCP()
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, false, nil
		}
		return nil, false, err
	}
	var stream net.Conn = conn
	tlsEnabled := false
	if l.TLSConfig != nil {
		tlsConn := tls.Server(conn, l.TLSConfig.Clone())
		if timeout >= 0 {
			_ = tlsConn.SetDeadline(time.Now().Add(timeout))
		}
		if err := tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, false, nil
			}
			return nil, false, err
		}
		_ = tlsConn.SetDeadline(time.Time{})
		stream = tlsConn
		tlsEnabled = true
	}
	return &NetStream{Conn: stream, TLS: tlsEnabled}, true, nil
}

// NetStream is the common byte-stream abstraction used by TCP and TLS.
type NetStream struct {
	Conn   net.Conn
	TLS    bool
	closed atomic.Bool
}

func (s *NetStream) Type() ObjectType { return NET_STREAM_OBJ }
func (s *NetStream) Inspect() string {
	if s == nil || s.Conn == nil {
		return "NetStream<nil>"
	}
	protocol := "tcp"
	if s.TLS {
		protocol = "tls"
	}
	return fmt.Sprintf("NetStream<%s %s -> %s>", protocol, s.Conn.LocalAddr(), s.Conn.RemoteAddr())
}
func (s *NetStream) Close() error {
	if s == nil || s.Conn == nil || !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	return s.Conn.Close()
}
func (s *NetStream) Closed() bool { return s == nil || s.closed.Load() }
func (s *NetStream) TCPConn() *net.TCPConn {
	if s == nil || s.Conn == nil {
		return nil
	}
	if conn, ok := s.Conn.(*net.TCPConn); ok {
		return conn
	}
	if provider, ok := s.Conn.(interface{ NetConn() net.Conn }); ok {
		if conn, ok := provider.NetConn().(*net.TCPConn); ok {
			return conn
		}
	}
	return nil
}

// UDPSocket represents a bound UDP endpoint.
type UDPSocket struct {
	Conn   *net.UDPConn
	closed atomic.Bool
}

func (u *UDPSocket) Type() ObjectType { return UDP_SOCKET_OBJ }
func (u *UDPSocket) Inspect() string {
	if u == nil || u.Conn == nil {
		return "UDPSocket<nil>"
	}
	return fmt.Sprintf("UDPSocket<%s>", u.Conn.LocalAddr())
}
func (u *UDPSocket) Close() error {
	if u == nil || u.Conn == nil || !u.closed.CompareAndSwap(false, true) {
		return nil
	}
	return u.Conn.Close()
}
func (u *UDPSocket) Closed() bool { return u == nil || u.closed.Load() }
