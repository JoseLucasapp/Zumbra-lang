package builtins

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"zumbra/binarydata"
	"zumbra/object"
)

const webSocketMaxMessage = 16 * 1024 * 1024

func webSocketArg(value object.Object, name string) (*object.WebSocket, *object.Error) {
	socket, ok := value.(*object.WebSocket)
	if !ok {
		return nil, NewError("%s expects WebSocket, got %s", name, value.Type())
	}
	return socket, nil
}

func webSocketAccept(key string) string {
	digest := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(digest[:])
}

func WebSocketUpgradeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("webSocketUpgrade expects 1 argument, got %d", len(args))
		}
		request, ok := args[0].(*object.HttpRequest)
		if !ok {
			return NewError("webSocketUpgrade expects HttpRequest, got %s", args[0].Type())
		}
		if request.Writer == nil || request.Request == nil {
			return NewError("webSocketUpgrade requires a live server request")
		}
		if !strings.EqualFold(request.Request.Header.Get("Upgrade"), "websocket") || !headerContainsToken(request.Request.Header, "Connection", "upgrade") {
			return NewError("webSocketUpgrade request is missing Upgrade: websocket")
		}
		version := request.Request.Header.Get("Sec-WebSocket-Version")
		if version != "13" {
			return NewError("webSocketUpgrade only supports RFC 6455 version 13")
		}
		key := strings.TrimSpace(request.Request.Header.Get("Sec-WebSocket-Key"))
		if key == "" {
			return NewError("webSocketUpgrade request is missing Sec-WebSocket-Key")
		}
		hijacker, ok := request.Writer.(http.Hijacker)
		if !ok {
			return NewError("webSocketUpgrade is not supported by this HTTP writer")
		}
		connection, buffered, err := hijacker.Hijack()
		if err != nil {
			return NewError("webSocketUpgrade: %s", err)
		}
		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + webSocketAccept(key) + "\r\n\r\n"
		if _, err := buffered.WriteString(response); err != nil {
			_ = connection.Close()
			return NewError("webSocketUpgrade write handshake: %s", err)
		}
		if err := buffered.Flush(); err != nil {
			_ = connection.Close()
			return NewError("webSocketUpgrade flush handshake: %s", err)
		}
		request.Hijacked.Store(true)
		return &object.WebSocket{Conn: connection, Reader: buffered.Reader, Client: false}
	}}
}

func headerContainsToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func WebSocketConnectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("webSocketConnect expects 3 arguments, got %d", len(args))
		}
		target, errObj := httpString(args[0], "webSocketConnect URL")
		if errObj != nil {
			return errObj
		}
		headers, errObj := httpDict(args[1], "webSocketConnect headers")
		if errObj != nil {
			return errObj
		}
		timeoutMs, errObj := httpInt(args[2], "webSocketConnect timeout")
		if errObj != nil {
			return errObj
		}
		parsed, err := url.Parse(target)
		if err != nil || (parsed.Scheme != "ws" && parsed.Scheme != "wss") {
			return NewError("webSocketConnect expects a ws:// or wss:// URL")
		}
		host := parsed.Hostname()
		port := parsed.Port()
		if port == "" {
			if parsed.Scheme == "wss" {
				port = "443"
			} else {
				port = "80"
			}
		}
		address := net.JoinHostPort(host, port)
		dialer := &net.Dialer{Timeout: time.Duration(timeoutMs) * time.Millisecond}
		var connection net.Conn
		if parsed.Scheme == "wss" {
			tlsConnection, dialErr := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
			if dialErr != nil {
				return NewError("webSocketConnect TLS: %s", dialErr)
			}
			connection = tlsConnection
		} else {
			plain, dialErr := dialer.Dial("tcp", address)
			if dialErr != nil {
				return NewError("webSocketConnect: %s", dialErr)
			}
			connection = plain
		}
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			_ = connection.Close()
			return NewError("webSocketConnect key: %s", err)
		}
		key := base64.StdEncoding.EncodeToString(nonce[:])
		requestPath := parsed.EscapedPath()
		if requestPath == "" {
			requestPath = "/"
		}
		if parsed.RawQuery != "" {
			requestPath += "?" + parsed.RawQuery
		}
		var request bytes.Buffer
		fmt.Fprintf(&request, "GET %s HTTP/1.1\r\n", requestPath)
		fmt.Fprintf(&request, "Host: %s\r\n", parsed.Host)
		request.WriteString("Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\n")
		fmt.Fprintf(&request, "Sec-WebSocket-Key: %s\r\n", key)
		for _, pair := range headers.Pairs {
			fmt.Fprintf(&request, "%s: %s\r\n", pair.Key.Inspect(), pair.Value.Inspect())
		}
		request.WriteString("\r\n")
		if _, err := connection.Write(request.Bytes()); err != nil {
			_ = connection.Close()
			return NewError("webSocketConnect handshake write: %s", err)
		}
		reader := bufio.NewReader(connection)
		response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
		if err != nil {
			_ = connection.Close()
			return NewError("webSocketConnect handshake: %s", err)
		}
		if response.StatusCode != http.StatusSwitchingProtocols || !strings.EqualFold(response.Header.Get("Upgrade"), "websocket") || !headerContainsToken(response.Header, "Connection", "upgrade") || response.Header.Get("Sec-WebSocket-Accept") != webSocketAccept(key) {
			_ = response.Body.Close()
			_ = connection.Close()
			return NewError("webSocketConnect server rejected the upgrade: %s", response.Status)
		}
		_ = response.Body.Close()
		_ = connection.SetDeadline(time.Time{})
		return &object.WebSocket{Conn: connection, Reader: reader, Client: true}
	}}
}

func WebSocketWriteBuiltin(opcode byte) *object.Builtin {
	name := "webSocketWriteText"
	if opcode == 0x2 {
		name = "webSocketWriteBinary"
	} else if opcode == 0x9 {
		name = "webSocketPing"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("%s expects 2 arguments, got %d", name, len(args))
		}
		socket, errObj := webSocketArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		var payload []byte
		if opcode == 0x1 || opcode == 0x9 {
			text, errObj := httpString(args[1], name)
			if errObj != nil {
				return errObj
			}
			payload = []byte(text)
		} else {
			data, err := binarydata.Bytes(args[1])
			if err != nil {
				return NewError("%s expects a byte-compatible buffer", name)
			}
			payload = data
		}
		if opcode >= 0x8 && len(payload) > 125 {
			return NewError("WebSocket control frames cannot exceed 125 bytes")
		}
		if err := webSocketWriteFrame(socket, opcode, payload); err != nil {
			return NewError("%s: %s", name, err)
		}
		return &object.Integer{Value: int64(len(payload))}
	}}
}

func WebSocketReadBuiltin(withTimeout bool) *object.Builtin {
	name := "webSocketRead"
	expected := 1
	if withTimeout {
		name = "webSocketReadTimeout"
		expected = 2
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		socket, errObj := webSocketArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		if withTimeout {
			timeout, errObj := httpInt(args[1], name+" timeout")
			if errObj != nil {
				return errObj
			}
			_ = socket.Conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
			defer socket.Conn.SetReadDeadline(time.Time{})
		}
		messageType, payload, open, err := webSocketReadMessage(socket)
		if err != nil {
			if withTimeout {
				if networkError, ok := err.(net.Error); ok && networkError.Timeout() {
					return &object.Array{Elements: []object.Object{&object.String{Value: ""}, &object.Null{}, &object.Boolean{Value: true}, &object.Boolean{Value: false}}}
				}
			}
			return NewError("%s: %s", name, err)
		}
		var data object.Object = &object.Null{}
		if messageType == "text" {
			data = &object.String{Value: string(payload)}
		} else if messageType != "close" {
			buffer := &object.ByteArray{Data: make([]byte, len(payload))}
			copy(buffer.Data, payload)
			data = buffer
		}
		elements := []object.Object{&object.String{Value: messageType}, data, &object.Boolean{Value: open}}
		if withTimeout {
			elements = append(elements, &object.Boolean{Value: true})
		}
		return &object.Array{Elements: elements}
	}}
}

func WebSocketCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("webSocketClose expects 3 arguments, got %d", len(args))
		}
		socket, errObj := webSocketArg(args[0], "webSocketClose")
		if errObj != nil {
			return errObj
		}
		code, errObj := httpInt(args[1], "webSocketClose code")
		if errObj != nil {
			return errObj
		}
		reason, errObj := httpString(args[2], "webSocketClose reason")
		if errObj != nil {
			return errObj
		}
		payload := make([]byte, 2+len(reason))
		binary.BigEndian.PutUint16(payload[:2], uint16(code))
		copy(payload[2:], reason)
		socket.CloseOnce.Do(func() {
			_ = webSocketWriteFrame(socket, 0x8, payload)
			socket.Closed.Store(true)
			_ = socket.Conn.Close()
		})
		return &object.Boolean{Value: true}
	}}
}

func WebSocketClosedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("webSocketClosed expects 1 argument, got %d", len(args))
		}
		socket, errObj := webSocketArg(args[0], "webSocketClosed")
		if errObj != nil {
			return errObj
		}
		return &object.Boolean{Value: socket.Closed.Load()}
	}}
}

func webSocketWriteFrame(socket *object.WebSocket, opcode byte, payload []byte) error {
	if socket == nil || socket.Conn == nil || socket.Closed.Load() {
		return errors.New("WebSocket is closed")
	}
	if len(payload) > webSocketMaxMessage {
		return fmt.Errorf("WebSocket message exceeds %d bytes", webSocketMaxMessage)
	}
	socket.WriteMu.Lock()
	defer socket.WriteMu.Unlock()
	header := []byte{0x80 | opcode}
	maskBit := byte(0)
	if socket.Client {
		maskBit = 0x80
	}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, maskBit|byte(length))
	case length <= 65535:
		header = append(header, maskBit|126, byte(length>>8), byte(length))
	default:
		header = append(header, maskBit|127)
		size := uint64(length)
		for shift := 56; shift >= 0; shift -= 8 {
			header = append(header, byte(size>>shift))
		}
	}
	data := payload
	if socket.Client {
		var key [4]byte
		if _, err := rand.Read(key[:]); err != nil {
			return err
		}
		header = append(header, key[:]...)
		data = append([]byte(nil), payload...)
		for index := range data {
			data[index] ^= key[index%4]
		}
	}
	if _, err := socket.Conn.Write(header); err != nil {
		return err
	}
	_, err := socket.Conn.Write(data)
	return err
}

func webSocketReadMessage(socket *object.WebSocket) (string, []byte, bool, error) {
	if socket == nil || socket.Conn == nil || socket.Closed.Load() {
		return "close", nil, false, nil
	}
	socket.ReadMu.Lock()
	defer socket.ReadMu.Unlock()
	reader := socket.Reader
	if reader == nil {
		reader = bufio.NewReader(socket.Conn)
		socket.Reader = reader
	}
	var accumulated []byte
	messageOpcode := byte(0)
	for {
		first, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				socket.Closed.Store(true)
				return "close", nil, false, nil
			}
			return "", nil, true, err
		}
		second, err := reader.ReadByte()
		if err != nil {
			return "", nil, true, err
		}
		final := first&0x80 != 0
		opcode := first & 0x0f
		masked := second&0x80 != 0
		length := uint64(second & 0x7f)
		if length == 126 {
			var extended [2]byte
			if _, err := io.ReadFull(reader, extended[:]); err != nil {
				return "", nil, true, err
			}
			length = uint64(binary.BigEndian.Uint16(extended[:]))
		} else if length == 127 {
			var extended [8]byte
			if _, err := io.ReadFull(reader, extended[:]); err != nil {
				return "", nil, true, err
			}
			length = binary.BigEndian.Uint64(extended[:])
		}
		if length > webSocketMaxMessage || uint64(len(accumulated))+length > webSocketMaxMessage {
			return "", nil, true, errors.New("WebSocket message exceeds safety limit")
		}
		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(reader, mask[:]); err != nil {
				return "", nil, true, err
			}
		}
		payload := make([]byte, int(length))
		if _, err := io.ReadFull(reader, payload); err != nil {
			return "", nil, true, err
		}
		if masked {
			for index := range payload {
				payload[index] ^= mask[index%4]
			}
		}
		if opcode >= 0x8 && !final {
			return "", nil, true, errors.New("fragmented WebSocket control frame")
		}
		switch opcode {
		case 0x0:
			if messageOpcode == 0 {
				return "", nil, true, errors.New("unexpected continuation frame")
			}
			accumulated = append(accumulated, payload...)
		case 0x1, 0x2:
			if messageOpcode != 0 {
				return "", nil, true, errors.New("new WebSocket message before continuation completed")
			}
			messageOpcode = opcode
			accumulated = append(accumulated, payload...)
		case 0x8:
			socket.Closed.Store(true)
			_ = webSocketWriteFrame(socket, 0x8, payload)
			_ = socket.Conn.Close()
			return "close", payload, false, nil
		case 0x9:
			_ = webSocketWriteFrame(socket, 0xA, payload)
			return "ping", payload, true, nil
		case 0xA:
			return "pong", payload, true, nil
		default:
			return "", nil, true, fmt.Errorf("unsupported WebSocket opcode %d", opcode)
		}
		if final {
			if messageOpcode == 0x1 {
				return "text", accumulated, true, nil
			}
			return "binary", accumulated, true, nil
		}
	}
}

// Keep strconv imported on all supported Go versions where response parsing
// may optimize the import away under build tags.
var _ = strconv.IntSize
