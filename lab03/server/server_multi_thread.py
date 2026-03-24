import os
import socketserver
import sys
import threading
import time
from urllib.parse import unquote


def build_response(status_line: str, body: bytes, content_type: str = "text/plain; charset=utf-8") -> bytes:
    headers = [
        status_line,
        "Connection: close",
        f"Content-Length: {len(body)}",
        f"Content-Type: {content_type}",
        "",
        "",
    ]
    return "\r\n".join(headers).encode("utf-8") + body


def guess_content_type(path: str) -> str:
    ext = os.path.splitext(path)[1].lower()
    if ext in (".html", ".htm"):
        return "text/html; charset=utf-8"
    if ext == ".txt":
        return "text/plain; charset=utf-8"
    if ext == ".css":
        return "text/css; charset=utf-8"
    if ext == ".js":
        return "application/javascript; charset=utf-8"
    if ext == ".json":
        return "application/json; charset=utf-8"
    if ext == ".png":
        return "image/png"
    if ext in (".jpg", ".jpeg"):
        return "image/jpeg"
    if ext == ".gif":
        return "image/gif"
    return "application/octet-stream"


def parse_request_path(request_text: str) -> str:
    first_line = request_text.split("\r\n", 1)[0]
    parts = first_line.split()
    if len(parts) < 3:
        raise ValueError("Bad request line")

    method, raw_path, _ = parts[0], parts[1], parts[2]
    if method != "GET":
        raise PermissionError("Method not allowed")

    path = unquote(raw_path.split("?", 1)[0])
    if path == "/":
        path = "/index.html"
    return path


def safe_local_path(root_dir: str, request_path: str) -> str:
    relative = request_path.lstrip("/")
    abs_path = os.path.abspath(os.path.join(root_dir, relative))
    root_abs = os.path.abspath(root_dir)
    if not abs_path.startswith(root_abs):
        raise ValueError("Path traversal detected")
    return abs_path


class HttpFileHandler(socketserver.BaseRequestHandler):

    def handle(self) -> None:
        root_dir = self.server.root_dir
        client_addr = self.client_address
        semaphore = self.server._semaphore

        semaphore.acquire()
        
        try:
            request_bytes = self.request.recv(4096)
            if not request_bytes:
                return

            request_text = request_bytes.decode("utf-8", errors="replace")
            request_path = parse_request_path(request_text)
            local_path = safe_local_path(root_dir, request_path)

            if not os.path.isfile(local_path):
                body = b"404 Not Found"
                response = build_response("HTTP/1.1 404 Not Found", body)
            else:
                with open(local_path, "rb") as file:
                    body = file.read()
                content_type = guess_content_type(local_path)
                response = build_response("HTTP/1.1 200 OK", body, content_type=content_type)

        except PermissionError:
            body = b"405 Method Not Allowed"
            response = build_response("HTTP/1.1 405 Method Not Allowed", body)
        except ValueError:
            body = b"400 Bad Request"
            response = build_response("HTTP/1.1 400 Bad Request", body)
        except Exception as error:
            body = f"500 Internal Server Error\n{error}".encode("utf-8", errors="replace")
            response = build_response("HTTP/1.1 500 Internal Server Error", body)
        else:
            print(f"[{threading.current_thread().name}] Start sending response to {client_addr}")
            time.sleep(4) # задержка для демонстрации многопоточности
            self.request.sendall(response)
            print(f"[{threading.current_thread().name}] Response sent to {client_addr}")
        finally:
            semaphore.release()


class ThreadingTCPServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True

    def __init__(self, server_address, handler_cls, concurrency_level: int):
        super().__init__(server_address, handler_cls)
        self._semaphore = threading.Semaphore(concurrency_level)



def main() -> None:
    if len(sys.argv) < 2 or len(sys.argv) > 3:
        print("Usage: python server_multi_thread.py <server_port> <concurrency_level>")
        sys.exit(1)

    try:
        port = int(sys.argv[1])
    except ValueError:
        print("Port must be integer.")
        sys.exit(1)

    if len(sys.argv) == 3:
        try:
            concurrency_level = int(sys.argv[2])
        except ValueError:
            print("Сoncurrency level must be integer.")
            sys.exit(1)

        if concurrency_level <= 0:
            print("concurrency_level must be greater than 0.")
            sys.exit(1)
    else:
        concurrency_level = 10

    host = "127.0.0.1"
    root_dir = os.getcwd()

    with ThreadingTCPServer((host, port), HttpFileHandler, concurrency_level) as server:
        server.root_dir = root_dir
        print(f"Multithreaded server started: http://{host}:{port}")
        print(f"Serving files from: {root_dir}")
        print(f"Concurrency limit: {concurrency_level}")
        server.serve_forever()


if __name__ == "__main__":
    main()
