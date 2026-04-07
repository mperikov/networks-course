import argparse
import socket


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="UDP broadcast client: receives server time messages."
    )
    parser.add_argument(
        "--host",
        default="0.0.0.0",
        help="Local bind address (default: 0.0.0.0)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=5005,
        help="UDP port to listen on (default: 5005)",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind((args.host, args.port))

    print(f"UDP broadcast client listening on {args.host}:{args.port}")
    try:
        while True:
            data, addr = sock.recvfrom(4096)
            message = data.decode("utf-8", errors="replace")
            print(f"Received from {addr[0]}:{addr[1]} -> {message}")
    except KeyboardInterrupt:
        print("\nClient stopped.")
    finally:
        sock.close()


if __name__ == "__main__":
    main()
