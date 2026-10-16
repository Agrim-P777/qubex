#!/usr/bin/env python3
"""Real PTY smoke tests: independent JSON stdin, rendering and terminal cleanup."""
import errno
import fcntl
import json
import os
import pathlib
import pty
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time

BINARY = str(pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "bin/qbx").resolve())

def session(config, payload, interrupt=False, workflow=False, terminate=False):
    read_fd, write_fd = os.pipe()
    os.write(write_fd, payload.encode())
    os.close(write_fd)
    pid, master = pty.fork()
    if pid == 0:
        os.dup2(read_fd, 0)
        os.close(read_fd)
        env = dict(os.environ, TERM="xterm-256color", XDG_CONFIG_HOME=config, NO_COLOR="1")
        os.execve(BINARY, [BINARY], env)
    os.close(read_fd)
    fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", 28, 110, 0, 0))
    output = bytearray()
    def pump(seconds=0.2):
        until = time.monotonic() + seconds
        while time.monotonic() < until:
            if select.select([master], [], [], min(0.05, max(0, until-time.monotonic())))[0]:
                try:
                    chunk = os.read(master, 65536)
                except OSError as error:
                    if error.errno == errno.EIO:
                        return
                    raise
                if not chunk:
                    return
                output.extend(chunk)
                # Reply to terminal cursor/device/background queries if asked.
                if b"\x1b[6n" in chunk:
                    os.write(master, b"\x1b[1;1R")
                if b"\x1b[c" in chunk:
                    os.write(master, b"\x1b[?1;2c")
                if b"\x1b]11;?" in chunk:
                    os.write(master, b"\x1b]11;rgb:0000/0000/0000\x1b\\")
    try:
        pump(2)
        assert b"QUBEX" in output, output[-2000:]
        def keys(data, pause=0.25):
            os.write(master, data)
            pump(pause)
        if workflow:
            keys(b"j")
            keys(b"?")
            keys(b"/Iterate elements")
            keys(b"\r")
            for _ in range(4):
                keys(b"l")
            keys(b"f")
            keys(b"\r")
            keys(b"?")
            keys(b"/Select multiple")
            keys(b"\r")
            keys(b"\x1b[B \x1b[B ")
            keys(b"\r")
            keys(b"o")
            keys(b"/replicas\t\x1b[C")
            keys(b"\r")
            keys(b"c")
        if not interrupt:
            assert b"SOURCE" in output and b"JQ" in output, output[-2000:]
            os.write(master, b"?")
            pump()
            assert b"Operations" in output, output[-2000:]
            os.write(master, b"\x1b")
            pump()
            os.write(master, b"s")
            pump()
            os.write(master, (b"Acceptance services\r" if workflow else b"PTY saved query\r"))
            pump(0.4)
            fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", 20, 60, 0, 0))
            os.kill(pid, signal.SIGWINCH)
            pump()
        if terminate:
            os.kill(pid, signal.SIGTERM)
        else:
            os.write(master, b"\x03" if interrupt else b"q")
        pump(1)
        status = None
        for _ in range(30):
            done, status = os.waitpid(pid, os.WNOHANG)
            if done:
                break
            pump(0.1)
        assert done, "TUI did not terminate"
        code = os.waitstatus_to_exitcode(status)
        assert code == (130 if interrupt else 0), (code, output[-2000:])
        assert b"\x1b[?1049l" in output, "alternate screen was not restored"
        attrs = termios.tcgetattr(master)
        assert attrs[3] & termios.ICANON, "terminal remained noncanonical"
        return bytes(output)
    finally:
        try:
            os.kill(pid, signal.SIGKILL)
            os.waitpid(pid, 0)
        except ProcessLookupError:
            pass
        except ChildProcessError:
            pass
        os.close(master)

with tempfile.TemporaryDirectory(prefix="qubex-pty-") as config:
    transcript = session(config, '{"name":"api","items":[1,2,3]}')
    saved = json.loads((pathlib.Path(config)/"qubex/history.json").read_text())
    assert saved["records"][0]["name"] == "PTY saved query", (saved["records"][0]["name"],transcript[-1000:])
    assert saved["records"][0]["jq"] == "."
    session(config, '{"name":"api"}', interrupt=True)
    session(config, '{"name":"api"}', interrupt=True, terminate=True)
    payload=pathlib.Path('examples/services.json').read_text()
    session(config, payload, workflow=True)
    records=json.loads((pathlib.Path(config)/"qubex/history.json").read_text())["records"]
    record=records[-1]
    env=dict(os.environ,XDG_CONFIG_HOME=config)
    result=subprocess.run([BINARY,'history','run',str(record['id'])],input=payload,text=True,capture_output=True,env=env)
    assert result.returncode==0, result.stderr
    assert json.loads(result.stdout)==[{"name":"worker","replicas":8},{"name":"api","replicas":3}], (record['jq'],result.stdout)
    assert result.stdout.startswith('[\n  {\n') and '\x1b[' not in result.stdout, result.stdout
    by_name=subprocess.run([BINARY,'history','run',record['name']],input=payload,text=True,capture_output=True,env=env)
    assert by_name.returncode==0 and by_name.stdout==result.stdout, (by_name.returncode,by_name.stderr)

    master, slave=pty.openpty()
    try:
        color_env=dict(env,TERM='xterm-256color')
        color_env.pop('NO_COLOR',None)
        colored_run=subprocess.Popen([BINARY,'history','run',record['name']],stdin=subprocess.PIPE,stdout=slave,stderr=subprocess.PIPE,env=color_env)
        os.close(slave)
        slave=-1
        colored_run.stdin.write(payload.encode())
        colored_run.stdin.close()
        colored=bytearray()
        while True:
            try:
                chunk=os.read(master,65536)
            except OSError as error:
                if error.errno==errno.EIO:
                    break
                raise
            if not chunk:
                break
            colored.extend(chunk)
        assert colored_run.wait(timeout=5)==0, colored_run.stderr.read()
        assert b'\x1b[1;34m' in colored and b'\x1b[0;32m' in colored, colored[:300]
    finally:
        if slave!=-1:
            os.close(slave)
        os.close(master)
    print("PASS: piped input, independent keyboard, palette, save, resize, quit, Ctrl+C, SIGTERM and terminal restoration")
    print("PASS: keyboard-only filter → project → sort → copy → save → headless replay")
    print("PASS: named history run, pretty piped JSON and jq-style terminal colors")
