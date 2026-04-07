import argparse
import os
import socket
import smtplib
from email.message import EmailMessage


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Send e-mail in txt or html format via SMTP."
    )
    parser.add_argument("recipient", help="Recipient e-mail address")
    parser.add_argument(
        "--format",
        choices=("txt", "html"),
        default="txt",
        help="Message format: txt or html",
    )
    parser.add_argument(
        "--subject",
        default="Lab05 SMTP mail",
        help="Message subject",
    )
    parser.add_argument(
        "--body",
        default=None,
        help="Message body text. If omitted, a default message is used.",
    )
    parser.add_argument(
        "--body-file",
        default=None,
        help="Path to a file with message body (overrides --body).",
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
        help="SMTP login username (or set SMTP_USER env var)",
    )
    parser.add_argument(
        "--smtp-password",
        default=os.environ.get("SMTP_PASSWORD"),
        help="SMTP password/token (or set SMTP_PASSWORD env var)",
    )
    return parser.parse_args()


def get_body(args: argparse.Namespace) -> str:
    if args.body_file:
        with open(args.body_file, "r", encoding="utf-8") as f:
            return f.read()
    if args.body is not None:
        return args.body
    if args.format == "html":
        return (
            "<h2>Test</h2>"
            "<p>html message</p>"
        )
    return "Test: txt message"


def build_message(args: argparse.Namespace) -> EmailMessage:
    msg = EmailMessage()
    msg["From"] = args.smtp_user
    msg["To"] = args.recipient
    msg["Subject"] = args.subject

    body = get_body(args)
    subtype = "html" if args.format == "html" else "plain"
    msg.set_content(body, subtype=subtype, charset="utf-8")
    return msg


def send_email(args: argparse.Namespace) -> None:
    if not args.smtp_user or not args.smtp_password:
        raise ValueError(
            "SMTP credentials are required. "
            "Provide --smtp-user/--smtp-password or set SMTP_USER/SMTP_PASSWORD."
        )

    msg = build_message(args)

    try:
        with smtplib.SMTP(args.smtp_host, args.smtp_port, timeout=30) as server:
            server.ehlo()
            server.starttls()
            server.ehlo()
            server.login(args.smtp_user, args.smtp_password)
            server.send_message(msg)
    except socket.gaierror as exc:
        raise RuntimeError(
            f"Cannot resolve SMTP host '{args.smtp_host}'. "
            "Check SMTP_HOST/--smtp-host and DNS/network settings."
        ) from exc


def main() -> None:
    args = parse_args()
    send_email(args)
    print(
        f"Message sent successfully to {args.recipient} "
        f"in {args.format} format."
    )


if __name__ == "__main__":
    main()
