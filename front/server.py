#!/usr/bin/env python3

from __future__ import annotations

import http.server
import socketserver
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


PORT = 3000
ROOT = Path(__file__).resolve().parent


def is_local_host(hostname: str) -> bool:
    hostname = (hostname or "").strip().lower()
    return hostname in {"", "localhost", "127.0.0.1"}


def format_error(message: str) -> bytes:
    return message.encode("utf-8", errors="replace")


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(ROOT), **kwargs)

    def do_POST(self):
        parsed = urllib.parse.urlparse(self.path)
        if parsed.path != "/__upload_proxy":
            self.send_error(404, "Not found")
            return

        query = urllib.parse.parse_qs(parsed.query)
        upload_url = (query.get("uploadUrl") or [""])[0].strip()
        if not upload_url:
            self.send_response(400)
            self.end_headers()
            self.wfile.write(format_error("uploadUrl is required"))
            return

        body_length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(body_length)
        content_type = self.headers.get("X-Upload-Content-Type", "application/octet-stream").strip() or "application/octet-stream"

        try:
            status, payload = self.forward_upload(upload_url, body, content_type)
        except Exception as exc:
            self.send_response(502)
            self.end_headers()
            self.wfile.write(format_error(str(exc)))
            return

        self.send_response(status)
        self.end_headers()
        self.wfile.write(payload)

    def forward_upload(self, upload_url: str, body: bytes, content_type: str):
        parsed = urllib.parse.urlparse(upload_url)
        status, payload = self.try_upload(upload_url, body, content_type)
        if 200 <= status < 300:
            return status, payload

        if is_local_host(parsed.hostname):
            return status, payload

        fallback_netloc = f"localhost:{parsed.port or (443 if parsed.scheme == 'https' else 9000)}"
        fallback = parsed._replace(netloc=fallback_netloc)
        return self.try_upload(urllib.parse.urlunparse(fallback), body, content_type, host_header=parsed.netloc)

    def try_upload(self, target_url: str, body: bytes, content_type: str, host_header: str | None = None):
        request = urllib.request.Request(target_url, data=body, method="PUT")
        request.add_header("Content-Type", content_type)
        if host_header:
            request.add_header("Host", host_header)

        try:
            with urllib.request.urlopen(request) as response:
                return response.status, response.read()
        except urllib.error.HTTPError as error:
            return error.code, error.read()


class ThreadingHTTPServer(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True


def main():
    server = ThreadingHTTPServer(("0.0.0.0", PORT), Handler)
    print(f"front is serving on http://localhost:{PORT}")
    server.serve_forever()


if __name__ == "__main__":
    main()
