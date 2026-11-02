package tcp

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func startEcho(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				io.Copy(conn, conn)
			}()
		}
	}()
	return listener.Addr().String()
}

func startProxy(t *testing.T, target string) (string, func(), <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	errs := make(chan error, 1)
	go func() { errs <- RunTCPProxy(listener, target, nil, done) }()
	var once sync.Once
	stop := func() { once.Do(func() { close(done) }) }
	t.Cleanup(stop)
	return listener.Addr().String(), stop, errs
}

func TestTCPProxyRelaysData(t *testing.T) {
	proxyAddr, _, _ := startProxy(t, startEcho(t))

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("reply = %q, want ping", buffer)
	}
}

func TestTCPProxyHandlesParallelClients(t *testing.T) {
	proxyAddr, _, _ := startProxy(t, startEcho(t))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			payload := fmt.Sprintf("client-%d", i)
			conn, err := net.Dial("tcp", proxyAddr)
			if err != nil {
				t.Errorf("client %d dial: %v", i, err)
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			if _, err := conn.Write([]byte(payload)); err != nil {
				t.Errorf("client %d write: %v", i, err)
				return
			}
			buffer := make([]byte, len(payload))
			if _, err := io.ReadFull(conn, buffer); err != nil {
				t.Errorf("client %d read: %v", i, err)
				return
			}
			if string(buffer) != payload {
				t.Errorf("client %d reply = %q, want %q", i, buffer, payload)
			}
		}(i)
	}
	wg.Wait()
}

func TestTCPProxyClosesClientWhenUpstreamIsDown(t *testing.T) {
	down, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	target := down.Addr().String()
	down.Close()

	proxyAddr, _, _ := startProxy(t, target)

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	_, err = conn.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("expected the connection to be closed")
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		t.Fatal("proxy kept the client connection open instead of closing it")
	}
}

func TestTCPProxyStopsWhenDoneIsClosed(t *testing.T) {
	_, stop, errs := startProxy(t, startEcho(t))

	stop()

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("RunTCPProxy returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunTCPProxy did not stop")
	}
}
