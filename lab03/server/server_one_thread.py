import os
import socket
import sys
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


def handle_client(client_socket: socket.socket, root_dir: str) -> None:
    request_bytes = client_socket.recv(4096)
    if not request_bytes:
        return

    request_text = request_bytes.decode("utf-8", errors="replace")

    try:
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

    except FileNotFoundError:
        body = b"404 Not Found"
        response = build_response("HTTP/1.1 404 Not Found", body)
    except PermissionError:
        body = b"405 Method Not Allowed"
        response = build_response("HTTP/1.1 405 Method Not Allowed", body)
    except ValueError:
        body = b"400 Bad Request"
        response = build_response("HTTP/1.1 400 Bad Request", body)

    client_socket.sendall(response)


def main() -> None:
    if len(sys.argv) != 2:
        print("Usage: python server_one_thread.py <server_port>")
        sys.exit(1)

    try:
        port = int(sys.argv[1])
    except ValueError:
        print("Port must be an integer.")
        sys.exit(1)

    host = "127.0.0.1"
    root_dir = os.getcwd()

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
        server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server_socket.bind((host, port))
        server_socket.listen(1)
        print(f"Server started: http://{host}:{port}")
        print(f"Serving files from: {root_dir}")

        # Однопоточный режим: обрабатываем соединения последовательно.
        while True:
            client_socket, client_addr = server_socket.accept()
            with client_socket:
                print(f"Client connected: {client_addr}")
                handle_client(client_socket, root_dir)
                print("Response sent, connection closed.")


if __name__ == "__main__":
    main()
