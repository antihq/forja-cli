#!/usr/bin/env python3
"""Fake Forja API for verifying the forja CLI against a controlled backend.

Serves deterministic fixtures under /api/v1, requires bearer auth, logs every
request as one JSON line, and can inject a fixed failure for error-path cases.
Deployments created via POST /sites/<id>/deploy are kept in memory and show up
in later reads of /sites/<id>/deployments and /deployments/<id>. Sites created
via POST /servers/<id>/sites are kept in memory too, together with their
pending deployment (unless --deploy-key semantics apply), so a second read
proves the side effect. Deployment settings live per site as well:
GET /sites/<id>/settings returns the seeded or defaulted document in contract
key order, PATCH /sites/<id>/settings merges it for real (present fields
replace their stored value wholesale, absent fields stay, retention null or
out of 1-50 is a 422, unknown sites 404 like Laravel's model not found).

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

SETTINGS_KEY_ORDER = [
    "site_id", "zero_downtime_deployment", "deploy_notification_email",
    "deployment_releases_retention", "shared_directories", "shared_files",
    "writeable_directories", "hook_before_updating_repository",
    "hook_after_updating_repository", "hook_before_making_current",
    "hook_after_making_current",
]


def default_settings(site_id):
    return {
        "site_id": site_id,
        "zero_downtime_deployment": True,
        "deploy_notification_email": None,
        "deployment_releases_retention": 10,
        "shared_directories": ["storage"],
        "shared_files": [".env"],
        "writeable_directories": [],
        "hook_before_updating_repository": "",
        "hook_after_updating_repository": "",
        "hook_before_making_current": "",
        "hook_after_making_current": "",
    }


SETTINGS = {
    "site-01": default_settings("site-01"),
    "site-02": dict(default_settings("site-02"), deployment_releases_retention=30,
                    deploy_notification_email="ops@acme.test",
                    shared_directories=["storage", "public"]),
    "site-03": default_settings("site-03"),
}

lock = threading.Lock()
log_file = None
api_key = "forja-verify-key"
fail_status = None
fail_body = ""
next_dep_number = 200
next_site_number = 4


def now():
    return datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        self._handle()

    def do_POST(self):
        self._handle()

    def do_PATCH(self):
        self._handle()

    def _send(self, status, payload):
        data = payload if isinstance(payload, bytes) else json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _create_site(self, server_id, body):
        global next_site_number, next_dep_number

        server = next((s for s in SERVERS if s["id"] == server_id), None)
        if server is None:
            return self._send(404, {"message": "Server not found."})

        try:
            data = json.loads(body) if body else {}
        except ValueError:
            data = {}
        if not isinstance(data, dict):
            data = {}

        errors = {}
        address = data.get("address", "")
        if not address:
            errors["address"] = ["El campo address es obligatorio."]
        elif any(s.get("address", s.get("name")) == address for s in SITES):
            errors["address"] = ["El campo address ya ha sido tomado."]
        if not data.get("php_version"):
            errors["php_version"] = ["El campo php version es obligatorio."]
        if not data.get("type"):
            errors["type"] = ["El campo type es obligatorio."]
        if errors:
            fields = sorted(errors)
            message = errors[fields[0]][0]
            if len(fields) > 1:
                message += " (and %d more errors)" % (len(fields) - 1)
            return self._send(422, {"message": message, "errors": errors})

        with lock:
            site_id = "site-%02d" % next_site_number
            next_site_number += 1
            deployment = None
            if not data.get("use_deploy_key"):
                deployment = {
                    "id": "dep-%d" % next_dep_number,
                    "site_id": site_id,
                    "status": "pending",
                    "commit": None,
                    "created_at": now(),
                }
                next_dep_number += 1

        site = {
            "id": site_id,
            "server_id": server["id"],
            "address": address,
            "url": "https://" + address,
            "type": data["type"],
            "php_version": data["php_version"],
            "tls_setting": "auto",
            "repository_url": data.get("repository_url") or None,
            "repository_branch": data.get("repository_branch") or None,
            "zero_downtime_deployment": bool(data.get("zero_downtime_deployment", True)),
            "installed_at": None,
            "created_at": now(),
        }
        if deployment is not None:
            site["deployment"] = deployment
        else:
            site["deploy_key_public"] = (
                "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF0rjaVerifyKey" + site_id + " forja-verify"
            )

        with lock:
            SITES.append(site)
            settings = default_settings(site_id)
            settings["zero_downtime_deployment"] = site["zero_downtime_deployment"]
            SETTINGS[site_id] = settings
            if deployment is not None:
                DEPLOYMENTS.setdefault(site_id, []).append(deployment)

        return self._send(201, site)

    def _settings_document(self, site_id):
        with lock:
            stored = SETTINGS.get(site_id, default_settings(site_id))
            return {key: stored.get(key) for key in SETTINGS_KEY_ORDER}

    def _update_settings(self, site_id, body):
        try:
            data = json.loads(body) if body else {}
        except ValueError:
            data = None
        if not isinstance(data, dict):
            return self._send(422, {"message": "The settings body must be a json object."})

        errors = {}
        if "deployment_releases_retention" in data:
            value = data["deployment_releases_retention"]
            if isinstance(value, bool) or not isinstance(value, int):
                errors["deployment_releases_retention"] = [
                    "El campo deployment releases retention debe ser un entero."]
            elif value < 1:
                errors["deployment_releases_retention"] = [
                    "El campo deployment releases retention debe ser al menos 1."]
            elif value > 50:
                errors["deployment_releases_retention"] = [
                    "El campo deployment releases retention no debe ser mayor que 50."]
        if errors:
            return self._send(422, {"message": errors["deployment_releases_retention"][0],
                                    "errors": errors})

        with lock:
            stored = SETTINGS.setdefault(site_id, default_settings(site_id))
            for key in ("deploy_notification_email", "shared_directories", "shared_files",
                        "writeable_directories", "hook_before_updating_repository",
                        "hook_after_updating_repository", "hook_before_making_current",
                        "hook_after_making_current", "deployment_releases_retention"):
                if key in data:
                    stored[key] = data[key]

        return self._send(200, self._settings_document(site_id))

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

        if len(rest) == 3 and rest[0] == "servers" and rest[2] == "sites" and method == "POST":
            return self._create_site(rest[1], body)

        if rest == ["sites"] and method == "GET":
            server_id = query.get("server_id", "")
            matches = [s for s in SITES if not server_id or s["server_id"] == server_id]
            return self._send(200, {"data": matches})

        if len(rest) >= 2 and rest[0] == "sites":
            if len(rest) == 3 and rest[2] == "settings" and method in ("GET", "PATCH"):
                if not any(s["id"] == rest[1] for s in SITES):
                    return self._send(404, {"message": "No query results for model [App\\Models\\Site] %s." % rest[1]})
                if method == "GET":
                    return self._send(200, self._settings_document(rest[1]))
                return self._update_settings(rest[1], body)

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
                    "created_at": now(),
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
