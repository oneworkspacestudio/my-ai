# SPDX-License-Identifier: AGPL-3.0-or-later
"""My AI's private, authenticated SearXNG process. No public listener or UI."""
import ctypes
import hmac
import json
import logging
import os
from pathlib import Path
import sys
import threading


def main():
    token = os.environ.pop("MYAI_SEARCH_PROCESS_TOKEN", "")
    if len(token) != 64:
        raise SystemExit(2)
    work = Path(os.environ["MYAI_SEARCH_WORK"])
    os.environ["TEMP"] = str(work)
    os.environ["TMP"] = str(work)
    os.environ["SEARXNG_SETTINGS_PATH"] = str(Path(__file__).with_name("settings.yml"))
    os.environ["SEARXNG_SECRET"] = token
    os.environ["SEARXNG_DEBUG"] = "false"
    os.environ["SEARXNG_PUBLIC_INSTANCE"] = "false"
    os.environ["SEARXNG_LIMITER"] = "false"
    # SearXNG's caches contain engine metadata, never account data. Keep them in
    # this process's own private directory, outside all editable projects.
    import tempfile
    tempfile.tempdir = str(work)
    logging.disable(logging.CRITICAL)
    import warnings
    warnings.filterwarnings("ignore")
    from waitress import create_server
    from searx.webapp import app

    def private_app(environ, start_response):
        if not hmac.compare_digest(environ.get("HTTP_AUTHORIZATION", ""), "Bearer " + token):
            start_response("404 Not Found", [("Content-Type", "text/plain")])
            return [b"Not found"]
        if environ.get("REQUEST_METHOD") != "GET":
            start_response("405 Method Not Allowed", [("Content-Type", "text/plain")])
            return [b"Method not allowed"]
        if environ.get("PATH_INFO") == "/healthz":
            start_response("200 OK", [("Content-Type", "application/json")])
            return [b'{"provider":"SearXNG","ready":true}']
        if environ.get("PATH_INFO") != "/search":
            start_response("404 Not Found", [("Content-Type", "text/plain")])
            return [b"Not found"]
        return app(environ, start_response)

    # A closed/crashed My AI must never leave a detached search server behind.
    from ctypes import wintypes
    kernel = ctypes.windll.kernel32
    kernel.OpenProcess.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
    kernel.OpenProcess.restype = wintypes.HANDLE
    kernel.WaitForSingleObject.argtypes = [wintypes.HANDLE, wintypes.DWORD]
    kernel.WaitForSingleObject.restype = wintypes.DWORD
    parent = kernel.OpenProcess(0x00100000, False, os.getppid())
    if not parent:
        raise SystemExit(3)
    def watch_parent():
        kernel.WaitForSingleObject(parent, 0xFFFFFFFF)
        os._exit(0)
    threading.Thread(target=watch_parent, daemon=True).start()
    server = create_server(private_app, host="127.0.0.1", port=0, threads=4,
                           clear_untrusted_proxy_headers=True,
                           max_request_header_size=16384, max_request_body_size=4096)
    print(json.dumps({"port": int(server.effective_port), "provider": "SearXNG"}), flush=True)
    server.run()


if __name__ == "__main__":
    try:
        main()
    except Exception:
        # Engine exceptions may contain a query or remote response. The Go
        # parent reports a safe, fixed error instead of exposing these details.
        raise SystemExit(1)
