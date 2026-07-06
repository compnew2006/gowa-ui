#!/usr/bin/env python3
"""Serve acp_guide.html and proxy LM Studio requests to avoid browser CORS issues.

Usage:
  LM_API_TOKEN='your-token' python3 serve_acp_with_proxy.py
Then open:
  http://127.0.0.1:8000/acp_guide.html
And set endpoint in the page to:
  http://127.0.0.1:8000/_lmstudio/api/v1
"""

from __future__ import annotations

import os
import pathlib
import urllib.error
import urllib.request
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer

HOST = os.environ.get("ACP_HOST", "127.0.0.1")
PORT = int(os.environ.get("ACP_PORT", "8000"))
LMSTUDIO_TARGET = os.environ.get("LMSTUDIO_TARGET", "http://127.0.0.1:1234").rstrip("/")
LM_API_TOKEN = os.environ.get("LM_API_TOKEN", "").strip()
PROXY_PREFIX = "/_lmstudio"


class ProxyHandler(SimpleHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self):
        if self.path.startswith(PROXY_PREFIX):
            self._proxy_request()
            return

        # Open guide by default.
        if self.path in ("/", ""):
            self.path = "/acp_guide.html"

        super().do_GET()

    def do_HEAD(self):
        if self.path.startswith(PROXY_PREFIX):
            self._proxy_request(send_body=False)
            return
        super().do_HEAD()

    def do_POST(self):
        if self.path.startswith(PROXY_PREFIX):
            self._proxy_request()
            return
        self.send_error(405, "Method Not Allowed")

    def do_OPTIONS(self):
        if self.path.startswith(PROXY_PREFIX):
            self.send_response(204)
            self.send_header("Access-Control-Allow-Origin", "*")
            self.send_header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
            self.send_header("Access-Control-Allow-Headers", "Authorization,Content-Type")
            self.send_header("Access-Control-Max-Age", "86400")
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        self.send_response(204)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def _proxy_request(self, send_body: bool = True):
        upstream_path = self.path[len(PROXY_PREFIX):]
        if not upstream_path.startswith("/"):
            upstream_path = "/" + upstream_path
        upstream_url = f"{LMSTUDIO_TARGET}{upstream_path}"

        content_len = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(content_len) if content_len > 0 else None

        req_headers = {}
        content_type = self.headers.get("Content-Type")
        if content_type:
            req_headers["Content-Type"] = content_type

        auth = self.headers.get("Authorization", "").strip()
        if auth:
            req_headers["Authorization"] = auth
        elif LM_API_TOKEN:
            req_headers["Authorization"] = f"Bearer {LM_API_TOKEN}"

        request = urllib.request.Request(
            upstream_url,
            data=body,
            headers=req_headers,
            method=self.command,
        )

        try:
            with urllib.request.urlopen(request, timeout=120) as resp:
                payload = resp.read() if send_body else b""
                self.send_response(resp.status)
                for key, value in resp.headers.items():
                    k = key.lower()
                    if k in {"transfer-encoding", "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailers", "upgrade"}:
                        continue
                    if k == "content-length":
                        continue
                    self.send_header(key, value)
                self.send_header("Access-Control-Allow-Origin", "*")
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                if send_body and payload:
                    self.wfile.write(payload)
        except urllib.error.HTTPError as e:
            payload = e.read() if send_body else b""
            self.send_response(e.code)
            self.send_header("Content-Type", e.headers.get_content_type() if e.headers else "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            if send_body and payload:
                self.wfile.write(payload)
        except Exception as e:  # noqa: BLE001
            message = str(e).encode("utf-8", errors="replace")
            self.send_response(502)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.send_header("Content-Length", str(len(message)))
            self.end_headers()
            if send_body and message:
                self.wfile.write(message)


def main():
    workspace = pathlib.Path(__file__).resolve().parent
    os.chdir(workspace)
    server = ThreadingHTTPServer((HOST, PORT), ProxyHandler)
    print(f"Serving ACP guide at: http://{HOST}:{PORT}/acp_guide.html")
    print(f"Proxy endpoint base: http://{HOST}:{PORT}{PROXY_PREFIX}/api/v1")
    print(f"LM Studio target: {LMSTUDIO_TARGET}")
    if LM_API_TOKEN:
        print("LM_API_TOKEN: provided via environment")
    else:
        print("LM_API_TOKEN: not set (you can still provide token in page field)")
    server.serve_forever()


if __name__ == "__main__":
    main()
