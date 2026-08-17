#!/usr/bin/env python3
"""Source RCON client. The game client can already send rcon; this drives the
server without one, from a shell or from CI.

    SRCDS_RCONPW=... python3 deploy/rcon.py "sm_ap_status"
"""

import os
import socket
import struct
import sys

SERVERDATA_AUTH = 3
SERVERDATA_EXECCOMMAND = 2

# A refused password still answers with the request id, so only the type separates it from success.
SERVERDATA_AUTH_RESPONSE = 2
SERVERDATA_RESPONSE_VALUE = 0

# 4096 is Valve's limit. The count cap is ours: an endless reply is a bug, not a long answer.
PACKET_BYTES_MAX = 4096
PACKET_COUNT_MAX = 128
TIMEOUT_SECONDS = 10

# The empty command sent behind every real one: the server answers in order, so its reply ends it.
TERMINATOR_OFFSET = 1000


def pack(request_id: int, kind: int, body: str) -> bytes:
    payload = struct.pack("<ii", request_id, kind) + body.encode("utf-8") + b"\x00\x00"
    return struct.pack("<i", len(payload)) + payload


def read_exactly(sock: socket.socket, count: int) -> bytes:
    chunks = []
    remaining = count
    while remaining > 0:
        chunk = sock.recv(remaining)
        if not chunk:
            raise ConnectionError("the server closed the connection")
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)


def read_packet(sock: socket.socket) -> tuple[int, int, str]:
    size = struct.unpack("<i", read_exactly(sock, 4))[0]
    if size < 10 or size > PACKET_BYTES_MAX:
        raise ValueError(f"the server sent a {size} byte packet")
    payload = read_exactly(sock, size)
    request_id, kind = struct.unpack("<ii", payload[:8])
    return request_id, kind, payload[8:-2].decode("utf-8", "replace")


def authenticate(sock: socket.socket, password: str) -> None:
    sock.sendall(pack(1, SERVERDATA_AUTH, password))
    for _ in range(PACKET_COUNT_MAX):
        request_id, kind, _ = read_packet(sock)
        if kind == SERVERDATA_RESPONSE_VALUE:
            continue
        if kind == SERVERDATA_AUTH_RESPONSE and request_id == 1:
            return
        raise PermissionError(
            "the server refused the rcon password. It reads SRCDS_RCONPW at "
            "boot, so a value changed since then needs 'make restart'."
        )
    raise ConnectionError("the server never answered the authentication")


def run(sock: socket.socket, request_id: int, command: str) -> str:
    terminator = request_id + TERMINATOR_OFFSET
    sock.sendall(pack(request_id, SERVERDATA_EXECCOMMAND, command))
    sock.sendall(pack(terminator, SERVERDATA_EXECCOMMAND, ""))

    reply = []
    for _ in range(PACKET_COUNT_MAX):
        answered_id, _, body = read_packet(sock)
        if answered_id == terminator:
            return "".join(reply).strip()
        reply.append(body)
    raise ConnectionError(f"the server never finished answering {command!r}")


def main() -> int:
    commands = sys.argv[1:]
    if not commands:
        print(f"usage: {sys.argv[0]} <command> [command ...]", file=sys.stderr)
        return 2

    password = os.environ.get("SRCDS_RCONPW", "")
    if not password:
        print("set SRCDS_RCONPW, the same value the server booted with", file=sys.stderr)
        return 2

    # .env leaves a setting empty rather than unset, and a get() default only covers the unset case.
    host = os.environ.get("SRCDS_RCON_HOST") or "127.0.0.1"
    port = int(os.environ.get("SRCDS_PORT") or "27015")

    try:
        with socket.create_connection((host, port), timeout=TIMEOUT_SECONDS) as sock:
            authenticate(sock, password)
            for offset, command in enumerate(commands):
                reply = run(sock, offset + 2, command)
                print(f"$ {command}")
                if reply:
                    print(reply)
    except (OSError, ValueError, PermissionError) as error:
        print(f"rcon: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
