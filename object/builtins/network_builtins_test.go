package builtins

import (
	"path/filepath"
	"testing"
	"time"

	"zumbra/object"
)

func requireNetworkValue(t *testing.T, value object.Object) object.Object {
	t.Helper()
	if err, ok := value.(*object.Error); ok {
		t.Fatalf("network builtin failed: %s", err.Message)
	}
	return value
}

func TestTCPStreamRoundTrip(t *testing.T) {
	listener := requireNetworkValue(t, TCPListenBuiltin().Fn(NewString("127.0.0.1"), NewInteger(0))).(*object.NetListener)
	port := requireNetworkValue(t, ListenerAddressBuiltin(true).Fn(listener)).(*object.Integer).Value
	done := make(chan struct{})
	go func() {
		defer close(done)
		stream := requireNetworkValue(t, ListenerAcceptBuiltin(false).Fn(listener)).(*object.NetStream)
		payload := requireNetworkValue(t, StreamReadBuiltin(true, false).Fn(stream, NewInteger(4)))
		requireNetworkValue(t, StreamWriteBuiltin(true).Fn(stream, payload))
		requireNetworkValue(t, StreamCloseBuiltin().Fn(stream))
	}()
	client := requireNetworkValue(t, TCPConnectBuiltin(true).Fn(NewString("127.0.0.1"), NewInteger(port), NewInteger(1000))).(*object.NetStream)
	requireNetworkValue(t, StreamWriteBuiltin(true).Fn(client, NewString("ping")))
	response := requireNetworkValue(t, StreamReadBuiltin(true, false).Fn(client, NewInteger(4))).(*object.ByteArray)
	if string(response.Data) != "ping" {
		t.Fatalf("unexpected TCP response %q", response.Data)
	}
	requireNetworkValue(t, StreamCloseBuiltin().Fn(client))
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TCP server did not finish")
	}
	requireNetworkValue(t, ListenerCloseBuiltin().Fn(listener))
}

func TestUDPAndDNSLocalOnly(t *testing.T) {
	receiver := requireNetworkValue(t, UDPBindBuiltin().Fn(NewString("127.0.0.1"), NewInteger(0))).(*object.UDPSocket)
	sender := requireNetworkValue(t, UDPBindBuiltin().Fn(NewString("127.0.0.1"), NewInteger(0))).(*object.UDPSocket)
	port := requireNetworkValue(t, UDPAddressBuiltin(true).Fn(receiver)).(*object.Integer).Value
	requireNetworkValue(t, UDPSendToBuiltin().Fn(sender, NewString("127.0.0.1"), NewInteger(port), NewString("ok")))
	packet := requireNetworkValue(t, UDPReceiveFromBuiltin(true).Fn(receiver, NewInteger(16), NewInteger(1000))).(*object.Array)
	if len(packet.Elements) != 4 || !packet.Elements[3].(*object.Boolean).Value {
		t.Fatalf("unexpected UDP timeout result: %s", packet.Inspect())
	}
	if string(packet.Elements[0].(*object.ByteArray).Data) != "ok" {
		t.Fatalf("unexpected UDP payload: %s", packet.Elements[0].Inspect())
	}
	addresses := requireNetworkValue(t, DNSLookupBuiltin(false).Fn(NewString("localhost"))).(*object.Array)
	if len(addresses.Elements) == 0 {
		t.Fatal("localhost DNS lookup returned no addresses")
	}
	requireNetworkValue(t, UDPCloseBuiltin().Fn(sender))
	requireNetworkValue(t, UDPCloseBuiltin().Fn(receiver))
}

func TestTLSStreamRoundTrip(t *testing.T) {
	certificate, err := filepath.Abs(filepath.Join("..", "..", "code_examples", "native", "z10_test_cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := filepath.Abs(filepath.Join("..", "..", "code_examples", "native", "z10_test_key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	listener := requireNetworkValue(t, TLSListenBuiltin().Fn(NewString("127.0.0.1"), NewInteger(0), NewString(certificate), NewString(key))).(*object.NetListener)
	port := requireNetworkValue(t, ListenerAddressBuiltin(true).Fn(listener)).(*object.Integer).Value
	done := make(chan struct{})
	go func() {
		defer close(done)
		stream := requireNetworkValue(t, ListenerAcceptBuiltin(false).Fn(listener)).(*object.NetStream)
		payload := requireNetworkValue(t, StreamReadBuiltin(true, false).Fn(stream, NewInteger(4)))
		requireNetworkValue(t, StreamWriteBuiltin(true).Fn(stream, payload))
		requireNetworkValue(t, StreamCloseBuiltin().Fn(stream))
	}()
	client := requireNetworkValue(t, TLSConnectBuiltin(true).Fn(NewString("127.0.0.1"), NewInteger(port), NewString("localhost"), NewBoolean(true), NewInteger(1000))).(*object.NetStream)
	requireNetworkValue(t, StreamWriteBuiltin(true).Fn(client, NewString("tls!")))
	response := requireNetworkValue(t, StreamReadBuiltin(true, false).Fn(client, NewInteger(4))).(*object.ByteArray)
	if string(response.Data) != "tls!" {
		t.Fatalf("unexpected TLS response %q", response.Data)
	}
	requireNetworkValue(t, StreamCloseBuiltin().Fn(client))
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TLS server did not finish")
	}
	requireNetworkValue(t, ListenerCloseBuiltin().Fn(listener))
}
