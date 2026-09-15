#!/usr/bin/env python3
"""Run a command under a PTY so the bridge sees a TTY for approvals."""
import os
import pty
import select
import signal
import sys


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: pty-run <cmd> [args...]", file=sys.stderr)
        return 2
    cmd = sys.argv[1:]
    pid, fd = pty.fork()
    if pid == 0:
        os.execvp(cmd[0], cmd)
        os._exit(127)
    try:
        while True:
            try:
                r, _, _ = select.select([fd], [], [], 0.2)
            except InterruptedError:
                continue
            if fd in r:
                try:
                    data = os.read(fd, 4096)
                except OSError:
                    break
                if not data:
                    break
                os.write(sys.stdout.fileno(), data)
            # Forward stdin (allow/deny commands) to the PTY.
            try:
                r2, _, _ = select.select([sys.stdin.fileno()], [], [], 0)
            except (InterruptedError, OSError):
                r2 = []
            if sys.stdin.fileno() in r2:
                try:
                    inp = os.read(sys.stdin.fileno(), 4096)
                except OSError:
                    inp = b""
                if inp:
                    os.write(fd, inp)
            # Detect child exit.
            wpid, status = os.waitpid(pid, os.WNOHANG)
            if wpid == pid:
                # Drain remaining output.
                while True:
                    try:
                        r, _, _ = select.select([fd], [], [], 0.1)
                        if fd not in r:
                            break
                        data = os.read(fd, 4096)
                        if not data:
                            break
                        os.write(sys.stdout.fileno(), data)
                    except OSError:
                        break
                if os.WIFEXITED(status):
                    return os.WEXITSTATUS(status)
                return 1
    except KeyboardInterrupt:
        try:
            os.kill(pid, signal.SIGTERM)
        except OSError:
            pass
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
