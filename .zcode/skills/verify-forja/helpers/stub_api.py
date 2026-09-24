#!/usr/bin/env python3
"""Fake Forja API for verifying the forja CLI against a controlled backend.

Serves deterministic fixtures under /api/v1, requires bearer auth, logs every
request as one JSON line, and can inject a fixed failure for error-path cases.
Deployments created via POST /sites/<id>/deploy are kept in memory and show up
in later reads of /sites/<id>/deployments and /deployments/<id>.

Usage:
  python3 stub_api.py --port 0 --api-key forja-verify-key --log /tmp/stub.log \
      [--fail-status 422 --fail-body '{"message":"Ya hay un deploy en curso."}']

Prints "READY port=<n>" on stdout once it is accepting connections. Bind is
127.0.0.1 only; SIGTERM stops it.
"""

import argparse
import datetime
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

ME = {
    "id": 1,
    "name": "Verify Agent",
    "email": "verify@example.test",
    "teams": [
        {"id": 1, "name": "Personal", "personal_team": True},
        {"id": 2, "name": "Acme", "personal_team": False},
    ],
}

SERVERS = [
    {"id": "srv-01", "name": "web-1", "ip": "203.0.113.10", "status": "running",
     "region": "nyc1", "created_at": "2026-01-15T10:00:00Z"},
    {"id": "srv-02", "name": "db-1", "ip": "203.0.113.20", "status": "running",
     "region": "nyc1", "created_at": "2026-02-01T09:30:00Z"},
]

SITES = [
    {"id": "site-01", "name": "acme.com", "server_id": "srv-01", "status": "live",
     "repository": "acme/site", "created_at": "2026-01-16T08:00:00Z"},
    {"id": "site-02", "name": "staging.acme.com", "server_id": "srv-01", "status": "live",
     "repository": "acme/site", "created_at": "2026-01-20T12:00:00Z"},
    {"id": "site-03", "name": "intranet.corp", "server_id": "srv-02", "status": "paused",
     "repository": "corp/intranet", "created_at": "2026-03-01T15:30:00Z"},
]

DEPLOYMENTS = {
    "site-01": [{"id": "dep-100", "site_id": "site-01", "status": "success",
                 "commit": "a1b2c3", "created_at": "2026-04-01T09:00:00Z"}],
    "site-02": [],
    "site-03": [{"id": "dep-105", "site_id": "site-03", "status": "pending",
                 "commit": None, "created_at": "2026-04-02T11:00:00Z"}],
}

lock = threading.Lock()
log_file = None
api_key = "forja-verify-key"
fail_status = None
fail_body = ""
next_dep_number = 200


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        self._handle()

    def do_POST(self):
        self._handle()

    def _send(self, status, payload):
        data = payload if isinstance(payload, bytes) else json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _handle(self):
        global next_dep_number
        parsed = urlparse(self.path)
        path = parsed.path
        parts = [p for p in path.split("/") if p]
        query = {k: v[0] for k, v in parse_qs(parsed.query).items()}

        body = None
        length = int(self.headers.get("Content-Length") or 0)
        if length:
            body = self.rfile.read(length).decode(errors="replace")

        record = {
            "time": datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="milliseconds"),
            "method": self.command,
            "path": path,
            "query": query,
            "authorization": self.headers.get("Authorization", ""),
            "team": self.headers.get("X-Forja-Team", ""),
            "accept": self.headers.get("Accept", ""),
            "body": body,
        }
        with lock:
            log_file.write(json.dumps(record) + "\n")
            log_file.flush()

        if path == "/api/v1/ping":
            return self._send(200, {"status": "ok"})

        if self.headers.get("Authorization", "") != "Bearer " + api_key:
            return self._send(401, {"message": "Unauthenticated."})

        if fail_status is not None:
            return self._send(fail_status, fail_body.encode())

        rest = parts[2:]  # strip ["api", "v1"]
        method = self.command

        if rest == ["me"] and method == "GET":
            return self._send(200, {"data": ME})

        if rest == ["servers"] and method == "GET":
            return self._send(200, {"data": SERVERS})

        if len(rest) == 2 and rest[0] == "servers" and method == "GET":
            server = next((s for s in SERVERS if s["id"] == rest[1]), None)
            if server is None:
                return self._send(404, {"message": "Server not found."})
            return self._send(200, server)

        if rest == ["sites"] and method == "GET":
            server_id = query.get("server_id", "")
            matches = [s for s in SITES if not server_id or s["server_id"] == server_id]
            return self._send(200, {"data": matches})

        if len(rest) >= 2 and rest[0] == "sites":
            site = next((s for s in SITES if s["id"] == rest[1]), None)
            if site is None:
                return self._send(404, {"message": "Site not found."})

            if len(rest) == 2 and method == "GET":
                return self._send(200, site)

            if len(rest) == 3 and rest[2] == "deploy" and method == "POST":
                deployment = {
                    "id": "dep-%d" % next_dep_number,
                    "site_id": site["id"],
                    "status": "pending",
                    "commit": None,
                    "created_at": datetime.datetime.now(datetime.timezone.utc)
                        .strftime("%Y-%m-%dT%H:%M:%SZ"),
                }
                next_dep_number += 1
                with lock:
                    DEPLOYMENTS.setdefault(site["id"], []).append(deployment)
                return self._send(200, deployment)

            if len(rest) == 3 and rest[2] == "deployments" and method == "GET":
                return self._send(200, {"data": DEPLOYMENTS.get(site["id"], [])})

        if len(rest) == 2 and rest[0] == "deployments" and method == "GET":
            for rows in DEPLOYMENTS.values():
                deployment = next((d for d in rows if d["id"] == rest[1]), None)
                if deployment is not None:
                    return self._send(200, deployment)
            return self._send(404, {"message": "Deployment not found."})

        return self._send(404, {"message": "Not found."})


def main():
    global log_file, api_key, fail_status, fail_body

    parser = argparse.ArgumentParser(description="Fake Forja API for CLI verification")
    parser.add_argument("--port", type=int, default=0, help="port to bind (0 picks a free one)")
    parser.add_argument("--api-key", default="forja-verify-key", help="bearer key requests must send")
    parser.add_argument("--log", required=True, help="request log path, one JSON line per request")
    parser.add_argument("--fail-status", type=int, default=None,
                        help="answer every authorized request with this status instead of routing")
    parser.add_argument("--fail-body", default="",
                        help="response body for --fail-status (JSON message or raw text)")
    args = parser.parse_args()

    api_key = args.api_key
    fail_status = args.fail_status
    fail_body = args.fail_body
    log_file = open(args.log, "a", encoding="utf-8")

    server = ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    print("READY port=%d" % server.server_address[1], flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
