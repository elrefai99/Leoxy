import socketserver
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

HTTP_PORTS = [9001, 9002]
TCP_PORT = 9101
UDP_PORT = 9102
HOST = "127.0.0.1"


def make_handler(name):
    class Handler(BaseHTTPRequestHandler):
        def drain_body(self):
            if self.headers.get("Transfer-Encoding", "").lower() == "chunked":
                while True:
                    size = int(self.rfile.readline().split(b";")[0].strip() or b"0", 16)
                    if size == 0:
                        self.rfile.readline()
                        return
                    self.rfile.read(size + 2)
            length = int(self.headers.get("Content-Length") or 0)
            if length:
                self.rfile.read(length)

        def respond(self):
            self.drain_body()
            payload = ("ok:" + name).encode()
            self.send_response(200)
            self.send_header("X-Served-By", name)
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        do_GET = respond
        do_POST = respond
        do_PUT = respond
        do_PATCH = respond
        do_DELETE = respond
        do_OPTIONS = respond

        def log_message(self, *args):
            pass

    return Handler


class TCPEcho(socketserver.BaseRequestHandler):
    def handle(self):
        while True:
            data = self.request.recv(4096)
            if not data:
                return
            self.request.sendall(data)


class UDPEcho(socketserver.BaseRequestHandler):
    def handle(self):
        data, sock = self.request
        sock.sendto(data, self.client_address)


class ReusableTCPServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True


def serve(server):
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return thread


def main():
    servers = []
    for port in HTTP_PORTS:
        server = ThreadingHTTPServer((HOST, port), make_handler("backend-" + str(port)))
        server.daemon_threads = True
        servers.append(server)
    servers.append(ReusableTCPServer((HOST, TCP_PORT), TCPEcho))
    servers.append(socketserver.UDPServer((HOST, UDP_PORT), UDPEcho))

    threads = [serve(server) for server in servers]
    print("upstreams ready: http", HTTP_PORTS, "tcp", TCP_PORT, "udp", UDP_PORT, flush=True)
    for thread in threads:
        thread.join()


if __name__ == "__main__":
    main()
