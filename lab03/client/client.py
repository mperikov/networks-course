import socket
import sys


def build_get_request(server_host: str, filename: str) -> bytes:
    path = filename if filename.startswith("/") else f"/{filename}"
    request_lines = [
        f"GET {path} HTTP/1.1",
        f"Host: {server_host}",
        "Connection: close",
        "",
        "",
    ]
    return "\r\n".join(request_lines).encode("utf-8")


def split_http_response(response: bytes) -> tuple[bytes, bytes]:
    separator = b"\r\n\r\n"
    if separator in response:
        header_bytes, body_bytes = response.split(separator, 1)
        return header_bytes, body_bytes
    return response, b""


def print_response(header_bytes: bytes, body_bytes: bytes) -> None:
    print("=== RESPONSE HEADERS ===")
    print(header_bytes.decode("iso-8859-1", errors="replace"))
    print()
    print("=== RESPONSE BODY ===")

    content_type = ""
    for line in header_bytes.decode("iso-8859-1", errors="replace").split("\r\n"):
        if line.lower().startswith("content-type:"):
            content_type = line.split(":", 1)[1].strip().lower()
            break

    if content_type.startswith("text/"):
        print(body_bytes.decode("utf-8", errors="replace"))
    else:
        print(f"[binary data: {len(body_bytes)} bytes]")


def main() -> None:
    if len(sys.argv) != 4:
        print("Usage: python client.py <server_host> <server_port> <filename>")
        sys.exit(1)

    server_host = sys.argv[1]
    try:
        server_port = int(sys.argv[2])
    except ValueError:
        print("server_port must be an integer.")
        sys.exit(1)

    filename = sys.argv[3]
    request = build_get_request(server_host, filename)

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client_socket:
        client_socket.connect((server_host, server_port))
        client_socket.sendall(request)

        response_chunks = []
        while True:
            chunk = client_socket.recv(4096)
            if not chunk:
                break
            response_chunks.append(chunk)

    raw_response = b"".join(response_chunks)
    headers, body = split_http_response(raw_response)
    print_response(headers, body)


if __name__ == "__main__":
    main()
