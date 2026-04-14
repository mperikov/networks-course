import argparse
from ftplib import FTP, all_errors

HOST = "ftp.dlptest.com"
PORT = 21
USER = "dlpuser"
PASSWORD = "rNrKYTX9g7z3RgJRmxWuGHbeu"
COMMAND = "list"
FILE = ""


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--command", default=COMMAND, help="list | upload | download")
    parser.add_argument("--file", default=FILE, help="File for upload/download")
    parser.add_argument("--host", default=HOST, help="FTP host")
    parser.add_argument("--port", type=int, default=PORT, help="FTP port")
    parser.add_argument("--password", default=PASSWORD, help="FTP password")
    parser.add_argument("--user", default=USER, help="FTP user")
    args = parser.parse_args()
    
    with FTP() as ftp:
        ftp.connect(args.host, args.port)
        ftp.login(args.user, args.password)

        if args.command == "list":
            ftp.retrlines("LIST")

        elif args.command == "upload":
            if not args.file:
                print("Set --file")
                return
            with open(args.file, "rb") as f:
                ftp.storbinary(f"STOR {args.file}", f)
            print(f"Uploaded: {args.file}")

        elif args.command == "download":
            if not args.file:
                print("Set --file")
                return
            with open(args.file, "wb") as f:
                ftp.retrbinary(f"RETR {args.file}", f.write)
            print(f"Downloaded: {args.file}")

        else:
            print("Unknown command")


if __name__ == "__main__":
    main()
