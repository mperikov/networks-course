import argparse
import base64
import os
import socket
import ssl

CRLF = "\r\n"

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Send plain text e-mail via SMTP using sockets."
    )
    parser.add_argument("recipient", help="Recipient e-mail address")
    parser.add_argument(
        "--subject",
        default="Lab05 SMTP socket client",
        help="Message subject",
    )
    parser.add_argument(
        "--body",
        default="Test text message sent via SMTP over sockets.",
        help="Plain text message body",
    )
    parser.add_argument(
        "--smtp-host",
        default=os.environ.get("SMTP_HOST", "smtp.mail.ru"),
        help="SMTP host. Default: SMTP_HOST or smtp.mail.ru",
    )
    parser.add_argument(
        "--smtp-port",
        type=int,
        default=int(os.environ.get("SMTP_PORT", "587")),
        help="SMTP port. Default: SMTP_PORT or 587",
    )
    parser.add_argument(
        "--smtp-user",
        default=os.environ.get("SMTP_USER"),
        help="SMTP login username (or set SMTP_USER)",
    )
    parser.add_argument(
        "--smtp-password",
        default=os.environ.get("SMTP_PASSWORD"),
        help="SMTP password/token (or set SMTP_PASSWORD)",
    )
    return parser.parse_args()


def recv_smtp_response(file_obj, expected_codes) -> str:
    lines = []
    while True:
        line = file_obj.readline()
        if not line:
            raise RuntimeError("SMTP server closed connection unexpectedly.")
        decoded = line.decode("utf-8", errors="replace").rstrip("\r\n")
        lines.append(decoded)
        if len(decoded) >= 4 and decoded[3] == " ":
            break

    code_text = lines[-1][:3]
    if not code_text.isdigit():
        raise RuntimeError(f"Malformed SMTP response: {lines[-1]}")

    code = int(code_text)
    if code not in expected_codes:
        joined = " | ".join(lines)
        raise RuntimeError(f"Unexpected SMTP response {code}: {joined}")
    return "\n".join(lines)


def send_smtp_command(sock, file_obj, command, expected_codes) -> str:
    sock.sendall((command + CRLF).encode("utf-8"))
    return recv_smtp_response(file_obj, expected_codes)


def build_message(sender: str, recipient: str, subject: str, body: str) -> str:
    escaped_lines = []
    for line in body.splitlines() or [""]:
        escaped_lines.append("." + line if line.startswith(".") else line)

    headers = [
        f"From: {sender}",
        f"To: {recipient}",
        f"Subject: {subject}",
        "MIME-Version: 1.0",
        "Content-Type: text/plain; charset=utf-8",
        "Content-Transfer-Encoding: 8bit",
        "",
    ]
    return CRLF.join(headers + escaped_lines)


def send_email_via_smtp_socket(args: argparse.Namespace) -> None:
    if not args.smtp_user or not args.smtp_password:
        raise ValueError(
            "SMTP credentials are required. "
            "Provide --smtp-user/--smtp-password or set SMTP_USER/SMTP_PASSWORD."
        )
    sender = args.smtp_user

    try:
        raw_sock = socket.create_connection((args.smtp_host, args.smtp_port), timeout=30)
    except socket.gaierror as exc:
        raise RuntimeError(
            f"Cannot resolve SMTP host '{args.smtp_host}'. "
            "Check SMTP_HOST/--smtp-host and DNS/network settings."
        ) from exc

    with raw_sock:
        raw_sock.settimeout(30)
        file_obj = raw_sock.makefile("rb")
        recv_smtp_response(file_obj, {220})

        hostname = socket.gethostname() or "localhost"
        send_smtp_command(raw_sock, file_obj, f"EHLO {hostname}", {250})
        send_smtp_command(raw_sock, file_obj, "STARTTLS", {220})

        context = ssl.create_default_context()
        tls_sock = context.wrap_socket(raw_sock, server_hostname=args.smtp_host)
        with tls_sock:
            tls_sock.settimeout(30)
            tls_file = tls_sock.makefile("rb")

            send_smtp_command(tls_sock, tls_file, f"EHLO {hostname}", {250})
            send_smtp_command(tls_sock, tls_file, "AUTH LOGIN", {334})
            send_smtp_command(
                tls_sock,
                tls_file,
                base64.b64encode(args.smtp_user.encode("utf-8")).decode("ascii"),
                {334},
            )
            send_smtp_command(
                tls_sock,
                tls_file,
                base64.b64encode(args.smtp_password.encode("utf-8")).decode("ascii"),
                {235},
            )

            send_smtp_command(
                tls_sock, tls_file, f"MAIL FROM:<{sender}>", {250}
            )
            send_smtp_command(
                tls_sock,
                tls_file,
                f"RCPT TO:<{args.recipient}>",
                {250, 251},
            )
            send_smtp_command(tls_sock, tls_file, "DATA", {354})

            message = build_message(sender, args.recipient, args.subject, args.body)
            tls_sock.sendall((message + CRLF + "." + CRLF).encode("utf-8"))
            recv_smtp_response(tls_file, {250})
            send_smtp_command(tls_sock, tls_file, "QUIT", {221})


def main() -> None:
    args = parse_args()
    send_email_via_smtp_socket(args)
    print(f"Message sent successfully to {args.recipient}.")


if __name__ == "__main__":
    main()
