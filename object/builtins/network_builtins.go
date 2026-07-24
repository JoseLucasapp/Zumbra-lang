package builtins

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"zumbra/binarydata"
	"zumbra/object"
)

func networkString(value object.Object, name string) (string, *object.Error) {
	text, ok := value.(*object.String)
	if !ok {
		return "", NewError("%s expects string, got %s", name, value.Type())
	}
	return text.Value, nil
}

func networkBool(value object.Object, name string) (bool, *object.Error) {
	boolean, ok := value.(*object.Boolean)
	if !ok {
		return false, NewError("%s expects bool, got %s", name, value.Type())
	}
	return boolean.Value, nil
}

func networkPort(value object.Object, name string) (int, *object.Error) {
	port, err := concurrencyInt(value, name)
	if err != nil {
		return 0, err
	}
	if port < 0 || port > 65535 {
		return 0, NewError("%s port must be between 0 and 65535, got %d", name, port)
	}
	return int(port), nil
}

func networkDuration(value object.Object, name string, allowNegative bool) (time.Duration, *object.Error) {
	milliseconds, err := concurrencyInt(value, name)
	if err != nil {
		return 0, err
	}
	if milliseconds < 0 && !allowNegative {
		return 0, NewError("%s timeout must be non-negative", name)
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}

func networkSize(value object.Object, name string) (int, *object.Error) {
	size, err := concurrencyInt(value, name)
	if err != nil {
		return 0, err
	}
	if size <= 0 {
		return 0, NewError("%s size must be greater than zero", name)
	}
	if size > 64*1024*1024 {
		return 0, NewError("%s size exceeds the 64 MiB safety limit", name)
	}
	return int(size), nil
}

func networkData(value object.Object, name string) ([]byte, *object.Error) {
	if text, ok := value.(*object.String); ok {
		return []byte(text.Value), nil
	}
	data, err := binarydata.Bytes(value)
	if err != nil {
		return nil, NewError("%s expects string or byte-compatible buffer: %s", name, err)
	}
	return data, nil
}

func netListenerArg(value object.Object, name string) (*object.NetListener, *object.Error) {
	listener, ok := value.(*object.NetListener)
	if !ok {
		return nil, NewError("%s expects NetListener, got %s", name, value.Type())
	}
	return listener, nil
}

func netStreamArg(value object.Object, name string) (*object.NetStream, *object.Error) {
	stream, ok := value.(*object.NetStream)
	if !ok {
		return nil, NewError("%s expects NetStream, got %s", name, value.Type())
	}
	return stream, nil
}

func udpSocketArg(value object.Object, name string) (*object.UDPSocket, *object.Error) {
	socket, ok := value.(*object.UDPSocket)
	if !ok {
		return nil, NewError("%s expects UDPSocket, got %s", name, value.Type())
	}
	return socket, nil
}

func splitAddress(address net.Addr) (string, int) {
	if address == nil {
		return "", 0
	}
	host, portText, err := net.SplitHostPort(address.String())
	if err != nil {
		return address.String(), 0
	}
	port, _ := strconv.Atoi(portText)
	return host, port
}

func listenTCP(host string, port int) (*net.TCPListener, error) {
	address, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	return net.ListenTCP("tcp", address)
}

func TCPListenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("tcpListen expects 2 arguments, got %d", len(args))
		}
		host, errObj := networkString(args[0], "tcpListen host")
		if errObj != nil {
			return errObj
		}
		port, errObj := networkPort(args[1], "tcpListen")
		if errObj != nil {
			return errObj
		}
		listener, err := listenTCP(host, port)
		if err != nil {
			return NewError("tcpListen %s:%d: %s", host, port, err)
		}
		return &object.NetListener{Listener: listener}
	}}
}

func TLSListenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("tlsListen expects 4 arguments, got %d", len(args))
		}
		host, errObj := networkString(args[0], "tlsListen host")
		if errObj != nil {
			return errObj
		}
		port, errObj := networkPort(args[1], "tlsListen")
		if errObj != nil {
			return errObj
		}
		certFile, errObj := networkString(args[2], "tlsListen certificate")
		if errObj != nil {
			return errObj
		}
		keyFile, errObj := networkString(args[3], "tlsListen key")
		if errObj != nil {
			return errObj
		}
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return NewError("tlsListen load certificate: %s", err)
		}
		listener, err := listenTCP(host, port)
		if err != nil {
			return NewError("tlsListen %s:%d: %s", host, port, err)
		}
		return &object.NetListener{Listener: listener, TLSConfig: &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}}
	}}
}

func tcpConnect(host string, port int, timeout time.Duration) (*object.NetStream, error) {
	dialer := net.Dialer{}
	if timeout >= 0 {
		dialer.Timeout = timeout
	}
	connection, err := dialer.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	return &object.NetStream{Conn: connection}, nil
}

func TCPConnectBuiltin(withTimeout bool) *object.Builtin {
	name := "tcpConnect"
	expected := 2
	if withTimeout {
		name = "tcpConnectTimeout"
		expected = 3
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		host, errObj := networkString(args[0], name+" host")
		if errObj != nil {
			return errObj
		}
		port, errObj := networkPort(args[1], name)
		if errObj != nil {
			return errObj
		}
		timeout := time.Duration(-1)
		if withTimeout {
			timeout, errObj = networkDuration(args[2], name, false)
			if errObj != nil {
				return errObj
			}
		}
		stream, err := tcpConnect(host, port, timeout)
		if err != nil {
			return NewError("%s %s:%d: %s", name, host, port, err)
		}
		return stream
	}}
}

func TLSConnectBuiltin(withTimeout bool) *object.Builtin {
	name := "tlsConnect"
	expected := 4
	if withTimeout {
		name = "tlsConnectTimeout"
		expected = 5
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		host, errObj := networkString(args[0], name+" host")
		if errObj != nil {
			return errObj
		}
		port, errObj := networkPort(args[1], name)
		if errObj != nil {
			return errObj
		}
		serverName, errObj := networkString(args[2], name+" serverName")
		if errObj != nil {
			return errObj
		}
		insecure, errObj := networkBool(args[3], name+" insecure")
		if errObj != nil {
			return errObj
		}
		timeout := time.Duration(-1)
		if withTimeout {
			timeout, errObj = networkDuration(args[4], name, false)
			if errObj != nil {
				return errObj
			}
		}
		if serverName == "" {
			serverName = host
		}
		dialer := &net.Dialer{}
		if timeout >= 0 {
			dialer.Timeout = timeout
		}
		connection, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, strconv.Itoa(port)), &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: insecure, // Explicit argument; never enabled implicitly.
			MinVersion:         tls.VersionTLS12,
		})
		if err != nil {
			return NewError("%s %s:%d: %s", name, host, port, err)
		}
		return &object.NetStream{Conn: connection, TLS: true}
	}}
}

func ListenerAcceptBuiltin(withTimeout bool) *object.Builtin {
	name := "listenerAccept"
	expected := 1
	if withTimeout {
		name = "listenerAcceptTimeout"
		expected = 2
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		listener, errObj := netListenerArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		timeout := time.Duration(-1)
		if withTimeout {
			timeout, errObj = networkDuration(args[1], name, false)
			if errObj != nil {
				return errObj
			}
		}
		stream, accepted, err := listener.Accept(timeout)
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		if withTimeout {
			value := object.Object(&object.Null{})
			if stream != nil {
				value = stream
			}
			return &object.Array{Elements: []object.Object{value, NewBoolean(accepted)}}
		}
		if !accepted || stream == nil {
			return NewError("%s did not accept a connection", name)
		}
		return stream
	}}
}

func ListenerCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("listenerClose expects 1 argument, got %d", len(args))
		}
		listener, errObj := netListenerArg(args[0], "listenerClose")
		if errObj != nil {
			return errObj
		}
		wasOpen := !listener.Closed()
		if err := listener.Close(); err != nil {
			return NewError("listenerClose: %s", err)
		}
		return NewBoolean(wasOpen)
	}}
}

func ListenerClosedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("listenerClosed expects 1 argument, got %d", len(args))
		}
		listener, errObj := netListenerArg(args[0], "listenerClosed")
		if errObj != nil {
			return errObj
		}
		return NewBoolean(listener.Closed())
	}}
}

func ListenerAddressBuiltin(port bool) *object.Builtin {
	name := "listenerAddress"
	if port {
		name = "listenerPort"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		listener, errObj := netListenerArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		if listener.Listener == nil {
			return NewError("%s listener is closed", name)
		}
		host, number := splitAddress(listener.Listener.Addr())
		if port {
			return NewInteger(int64(number))
		}
		return NewString(host)
	}}
}

func StreamReadBuiltin(exact bool, timeoutMode bool) *object.Builtin {
	name := "streamRead"
	expected := 2
	if exact {
		name = "streamReadExact"
	}
	if timeoutMode {
		name = "streamReadTimeout"
		expected = 3
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		stream, errObj := netStreamArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		size, errObj := networkSize(args[1], name)
		if errObj != nil {
			return errObj
		}
		if stream.Closed() || stream.Conn == nil {
			return NewError("%s cannot read a closed stream", name)
		}
		if timeoutMode {
			timeout, timeoutErr := networkDuration(args[2], name, false)
			if timeoutErr != nil {
				return timeoutErr
			}
			_ = stream.Conn.SetReadDeadline(time.Now().Add(timeout))
			defer stream.Conn.SetReadDeadline(time.Time{})
		}
		buffer := make([]byte, size)
		var count int
		var err error
		if exact {
			count, err = io.ReadFull(stream.Conn, buffer)
		} else {
			count, err = stream.Conn.Read(buffer)
		}
		buffer = buffer[:count]
		eof := errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		if timeoutMode {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return &object.Array{Elements: []object.Object{&object.ByteArray{Data: buffer}, NewBoolean(false), NewBoolean(false)}}
			}
			if err != nil && !eof {
				return NewError("%s: %s", name, err)
			}
			return &object.Array{Elements: []object.Object{&object.ByteArray{Data: buffer}, NewBoolean(count > 0), NewBoolean(eof)}}
		}
		if err != nil && !eof {
			return NewError("%s: %s", name, err)
		}
		if exact && count != size {
			return NewError("streamReadExact expected %d bytes, received %d", size, count)
		}
		return &object.ByteArray{Data: buffer}
	}}
}

func StreamWriteBuiltin(all bool) *object.Builtin {
	name := "streamWrite"
	if all {
		name = "streamWriteAll"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("%s expects 2 arguments, got %d", name, len(args))
		}
		stream, errObj := netStreamArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		data, errObj := networkData(args[1], name)
		if errObj != nil {
			return errObj
		}
		if stream.Closed() || stream.Conn == nil {
			return NewError("%s cannot write a closed stream", name)
		}
		written := 0
		for written < len(data) {
			count, err := stream.Conn.Write(data[written:])
			written += count
			if err != nil {
				return NewError("%s after %d bytes: %s", name, written, err)
			}
			if !all {
				break
			}
			if count == 0 {
				return NewError("%s made no progress", name)
			}
		}
		return NewInteger(int64(written))
	}}
}

func StreamCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("streamClose expects 1 argument, got %d", len(args))
		}
		stream, errObj := netStreamArg(args[0], "streamClose")
		if errObj != nil {
			return errObj
		}
		wasOpen := !stream.Closed()
		if err := stream.Close(); err != nil {
			return NewError("streamClose: %s", err)
		}
		return NewBoolean(wasOpen)
	}}
}

func StreamClosedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("streamClosed expects 1 argument, got %d", len(args))
		}
		stream, errObj := netStreamArg(args[0], "streamClosed")
		if errObj != nil {
			return errObj
		}
		return NewBoolean(stream.Closed())
	}}
}

func StreamShutdownBuiltin(write bool) *object.Builtin {
	name := "streamShutdownRead"
	if write {
		name = "streamShutdownWrite"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		stream, errObj := netStreamArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		connection := stream.TCPConn()
		if connection == nil {
			return NewError("%s requires a TCP-backed stream", name)
		}
		var err error
		if write {
			err = connection.CloseWrite()
		} else {
			err = connection.CloseRead()
		}
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		return NewBoolean(true)
	}}
}

func StreamAddressBuiltin(remote bool, port bool) *object.Builtin {
	name := "streamLocalAddress"
	if remote {
		name = "streamRemoteAddress"
	}
	if port {
		if remote {
			name = "streamRemotePort"
		} else {
			name = "streamLocalPort"
		}
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		stream, errObj := netStreamArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		var address net.Addr
		if remote {
			address = stream.Conn.RemoteAddr()
		} else {
			address = stream.Conn.LocalAddr()
		}
		host, number := splitAddress(address)
		if port {
			return NewInteger(int64(number))
		}
		return NewString(host)
	}}
}

func StreamSetTimeoutBuiltin(read bool) *object.Builtin {
	name := "streamSetWriteTimeout"
	if read {
		name = "streamSetReadTimeout"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("%s expects 2 arguments, got %d", name, len(args))
		}
		stream, errObj := netStreamArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		timeout, errObj := networkDuration(args[1], name, true)
		if errObj != nil {
			return errObj
		}
		deadline := time.Time{}
		if timeout >= 0 {
			deadline = time.Now().Add(timeout)
		}
		var err error
		if read {
			err = stream.Conn.SetReadDeadline(deadline)
		} else {
			err = stream.Conn.SetWriteDeadline(deadline)
		}
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		return &object.Null{}
	}}
}

func TCPSetKeepAliveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("tcpSetKeepAlive expects 3 arguments, got %d", len(args))
		}
		stream, errObj := netStreamArg(args[0], "tcpSetKeepAlive")
		if errObj != nil {
			return errObj
		}
		enabled, errObj := networkBool(args[1], "tcpSetKeepAlive enabled")
		if errObj != nil {
			return errObj
		}
		idle, errObj := networkDuration(args[2], "tcpSetKeepAlive idle", false)
		if errObj != nil {
			return errObj
		}
		connection := stream.TCPConn()
		if connection == nil {
			return NewError("tcpSetKeepAlive requires a TCP-backed stream")
		}
		if err := connection.SetKeepAlive(enabled); err != nil {
			return NewError("tcpSetKeepAlive: %s", err)
		}
		if enabled && idle > 0 {
			if err := connection.SetKeepAlivePeriod(idle); err != nil {
				return NewError("tcpSetKeepAlive period: %s", err)
			}
		}
		return &object.Null{}
	}}
}

func DNSLookupBuiltin(withTimeout bool) *object.Builtin {
	name := "dnsLookup"
	expected := 1
	if withTimeout {
		name = "dnsLookupTimeout"
		expected = 2
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		host, errObj := networkString(args[0], name+" host")
		if errObj != nil {
			return errObj
		}
		ctx := context.Background()
		cancel := func() {}
		if withTimeout {
			timeout, timeoutErr := networkDuration(args[1], name, false)
			if timeoutErr != nil {
				return timeoutErr
			}
			ctx, cancel = context.WithTimeout(ctx, timeout)
		}
		defer cancel()
		addresses, err := net.DefaultResolver.LookupHost(ctx, host)
		if withTimeout && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return &object.Array{Elements: []object.Object{&object.Array{}, NewBoolean(false)}}
		}
		if err != nil {
			return NewError("%s %q: %s", name, host, err)
		}
		values := make([]object.Object, 0, len(addresses))
		seen := map[string]bool{}
		for _, address := range addresses {
			if !seen[address] {
				seen[address] = true
				values = append(values, NewString(address))
			}
		}
		result := &object.Array{Elements: values}
		if withTimeout {
			return &object.Array{Elements: []object.Object{result, NewBoolean(true)}}
		}
		return result
	}}
}

func UDPBindBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("udpBind expects 2 arguments, got %d", len(args))
		}
		host, errObj := networkString(args[0], "udpBind host")
		if errObj != nil {
			return errObj
		}
		port, errObj := networkPort(args[1], "udpBind")
		if errObj != nil {
			return errObj
		}
		address, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return NewError("udpBind resolve: %s", err)
		}
		connection, err := net.ListenUDP("udp", address)
		if err != nil {
			return NewError("udpBind %s:%d: %s", host, port, err)
		}
		return &object.UDPSocket{Conn: connection}
	}}
}

func UDPSendToBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("udpSendTo expects 4 arguments, got %d", len(args))
		}
		socket, errObj := udpSocketArg(args[0], "udpSendTo")
		if errObj != nil {
			return errObj
		}
		host, errObj := networkString(args[1], "udpSendTo host")
		if errObj != nil {
			return errObj
		}
		port, errObj := networkPort(args[2], "udpSendTo")
		if errObj != nil {
			return errObj
		}
		data, errObj := networkData(args[3], "udpSendTo")
		if errObj != nil {
			return errObj
		}
		address, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return NewError("udpSendTo resolve: %s", err)
		}
		written, err := socket.Conn.WriteToUDP(data, address)
		if err != nil {
			return NewError("udpSendTo: %s", err)
		}
		return NewInteger(int64(written))
	}}
}

func UDPReceiveFromBuiltin(withTimeout bool) *object.Builtin {
	name := "udpReceiveFrom"
	expected := 2
	if withTimeout {
		name = "udpReceiveFromTimeout"
		expected = 3
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		socket, errObj := udpSocketArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		size, errObj := networkSize(args[1], name)
		if errObj != nil {
			return errObj
		}
		if withTimeout {
			timeout, timeoutErr := networkDuration(args[2], name, false)
			if timeoutErr != nil {
				return timeoutErr
			}
			_ = socket.Conn.SetReadDeadline(time.Now().Add(timeout))
			defer socket.Conn.SetReadDeadline(time.Time{})
		}
		buffer := make([]byte, size)
		count, address, err := socket.Conn.ReadFromUDP(buffer)
		if withTimeout {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return &object.Array{Elements: []object.Object{&object.ByteArray{}, NewString(""), NewInteger(0), NewBoolean(false)}}
			}
		}
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		result := []object.Object{&object.ByteArray{Data: buffer[:count]}, NewString(address.IP.String()), NewInteger(int64(address.Port))}
		if withTimeout {
			result = append(result, NewBoolean(true))
		}
		return &object.Array{Elements: result}
	}}
}

func UDPCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("udpClose expects 1 argument, got %d", len(args))
		}
		socket, errObj := udpSocketArg(args[0], "udpClose")
		if errObj != nil {
			return errObj
		}
		wasOpen := !socket.Closed()
		if err := socket.Close(); err != nil {
			return NewError("udpClose: %s", err)
		}
		return NewBoolean(wasOpen)
	}}
}

func UDPClosedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("udpClosed expects 1 argument, got %d", len(args))
		}
		socket, errObj := udpSocketArg(args[0], "udpClosed")
		if errObj != nil {
			return errObj
		}
		return NewBoolean(socket.Closed())
	}}
}

func UDPAddressBuiltin(port bool) *object.Builtin {
	name := "udpAddress"
	if port {
		name = "udpPort"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		socket, errObj := udpSocketArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		host, number := splitAddress(socket.Conn.LocalAddr())
		if port {
			return NewInteger(int64(number))
		}
		return NewString(host)
	}}
}

func networkError(name string, err error) *object.Error {
	return NewError("%s: %s", name, fmt.Sprint(err))
}
