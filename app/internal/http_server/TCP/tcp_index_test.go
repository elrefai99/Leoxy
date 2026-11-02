package tcp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"testing"
	"time"
)

func startEcho(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close echo listener: %v", err)
		}
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					t.Errorf("accept echo connection: %v", err)
				}
				return
			}
			go func() {
				defer func() {
					if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
						t.Errorf("close echo connection: %v", err)
					}
				}()
				if _, err := io.Copy(conn, conn); err != nil &&
					!errors.Is(err, net.ErrClosed) && !errors.Is(err, syscall.ECONNRESET) {
					t.Errorf("echo connection: %v", err)
				}
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
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close client connection: %v", err)
		}
	}()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}

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
			defer func() {
				if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
					t.Errorf("client %d close: %v", i, err)
				}
			}()
			if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Errorf("client %d set deadline: %v", i, err)
				return
			}
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
	if err := down.Close(); err != nil {
		t.Fatal(err)
	}

	proxyAddr, _, _ := startProxy(t, target)

	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close client connection: %v", err)
		}
	}()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

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
