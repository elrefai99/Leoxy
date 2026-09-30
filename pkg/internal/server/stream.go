package server

import (
	"io"
	"log"
	"net"
	"sync"
)

func RunTCPProxy(listener net.Listener, target string, done <-chan struct{}) error {
	defer listener.Close()
	go func() {
		<-done
		_ = listener.Close()
	}()

	for {
		client, err := listener.Accept()
		if err != nil {
			if isClosedError(err) {
				return nil
			}
			return err
		}
		go proxyTCPConnection(client, target)
	}
}

func proxyTCPConnection(client net.Conn, target string) {
	defer client.Close()
	upstream, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("TCP upstream %q connection failed: %v", target, err)
		return
	}
	defer upstream.Close()

	var copies sync.WaitGroup
	copies.Add(2)
	go func() {
		defer copies.Done()
		_, _ = io.Copy(upstream, client)
		if conn, ok := upstream.(*net.TCPConn); ok {
			_ = conn.CloseWrite()
		}
	}()
	go func() {
		defer copies.Done()
		_, _ = io.Copy(client, upstream)
		if conn, ok := client.(*net.TCPConn); ok {
			_ = conn.CloseWrite()
		}
	}()
	copies.Wait()
}

func RunUDPProxy(conn *net.UDPConn, target string, done <-chan struct{}) error {
	defer conn.Close()
	upstreamAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return err
	}

	var sessions sync.Map
	go func() {
		<-done
		_ = conn.Close()
		sessions.Range(func(_, value any) bool {
			_ = value.(*net.UDPConn).Close()
			return true
		})
	}()

	buffer := make([]byte, 65535)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if isClosedError(err) {
				return nil
			}
			return err
		}

		key := clientAddr.String()
		value, loaded := sessions.Load(key)
		var upstream *net.UDPConn
		if loaded {
			upstream = value.(*net.UDPConn)
		} else {
			upstream, err = net.DialUDP("udp", nil, upstreamAddr)
			if err != nil {
				log.Printf("UDP upstream %q connection failed: %v", target, err)
				continue
			}
			actual, loaded := sessions.LoadOrStore(key, upstream)
			if loaded {
				_ = upstream.Close()
				upstream = actual.(*net.UDPConn)
			} else {
				go relayUDPResponse(conn, upstream, clientAddr, func() {
					sessions.Delete(key)
				})
			}
		}
		if _, err := upstream.Write(buffer[:n]); err != nil {
			log.Printf("UDP upstream %q write failed: %v", target, err)
		}
	}
}

func relayUDPResponse(listener *net.UDPConn, upstream *net.UDPConn, client *net.UDPAddr, remove func()) {
	defer remove()
	defer upstream.Close()
	buffer := make([]byte, 65535)
	for {
		n, err := upstream.Read(buffer)
		if err != nil {
			return
		}
		if _, err := listener.WriteToUDP(buffer[:n], client); err != nil {
			return
		}
	}
}

func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*net.OpError)
	return ok && err.Error() == "use of closed network connection"
}
