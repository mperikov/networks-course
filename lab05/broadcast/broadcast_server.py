import argparse
import socket
import time
from datetime import datetime


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="UDP broadcast server: sends current time every second."
    )
    parser.add_argument(
        "--host",
        default="0.0.0.0",
        help="Broadcast address (default: 0.0.0.0)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=5005,
        help="UDP port for broadcast (default: 5005)",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    target = (args.host, args.port)

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)

    print(f"UDP broadcast server started on {args.host}:{args.port}")
    try:
        while True:
            current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            message = f"server_time={current_time}"
            sock.sendto(message.encode("utf-8"), target)
            print(f"Sent: {message}")
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nServer stopped.")
    finally:
        sock.close()


if __name__ == "__main__":
    main()
