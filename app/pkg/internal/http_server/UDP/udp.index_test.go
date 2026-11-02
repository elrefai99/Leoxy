package udp

import (
	"net"
	"sync"
	"testing"
	"time"
)

func startEcho(t *testing.T) string {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	go func() {
		buffer := make([]byte, 65535)
		for {
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			conn.WriteToUDP(buffer[:n], addr)
		}
	}()
	return conn.LocalAddr().String()
}

func startProxy(t *testing.T, target string) (*net.UDPAddr, func(), <-chan error) {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	errs := make(chan error, 1)
	go func() { errs <- RunUDPProxy(conn, target, nil, done) }()
	var once sync.Once
	stop := func() { once.Do(func() { close(done) }) }
	t.Cleanup(stop)
	return conn.LocalAddr().(*net.UDPAddr), stop, errs
}

func roundTrip(client *net.UDPConn, payload string) (string, error) {
	client.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := client.Write([]byte(payload)); err != nil {
		return "", err
	}
	buffer := make([]byte, 1024)
	n, err := client.Read(buffer)
	if err != nil {
		return "", err
	}
	return string(buffer[:n]), nil
}

func dial(t *testing.T, proxy *net.UDPAddr) *net.UDPConn {
	t.Helper()
	client, err := net.DialUDP("udp", nil, proxy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestUDPProxyRelaysDatagrams(t *testing.T) {
	proxy, _, _ := startProxy(t, startEcho(t))

	reply, err := roundTrip(dial(t, proxy), "ping")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ping" {
		t.Fatalf("reply = %q, want ping", reply)
	}
}

func TestUDPProxyKeepsOneSessionPerClient(t *testing.T) {
	proxy, _, _ := startProxy(t, startEcho(t))
	client := dial(t, proxy)

	for _, payload := range []string{"one", "two", "three"} {
		reply, err := roundTrip(client, payload)
		if err != nil {
			t.Fatal(err)
		}
		if reply != payload {
			t.Fatalf("reply = %q, want %q", reply, payload)
		}
	}
}

func TestUDPProxySeparatesClients(t *testing.T) {
	proxy, _, _ := startProxy(t, startEcho(t))
	first := dial(t, proxy)
	second := dial(t, proxy)

	replyA, err := roundTrip(first, "from-a")
	if err != nil {
		t.Fatal(err)
	}
	replyB, err := roundTrip(second, "from-b")
	if err != nil {
		t.Fatal(err)
	}
	if replyA != "from-a" || replyB != "from-b" {
		t.Fatalf("replies = %q and %q", replyA, replyB)
	}
}

func TestUDPProxyStopsWhenDoneIsClosed(t *testing.T) {
	_, stop, errs := startProxy(t, startEcho(t))

	stop()

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("RunUDPProxy returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunUDPProxy did not stop")
	}
}
