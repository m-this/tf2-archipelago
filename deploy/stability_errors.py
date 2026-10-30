"""Extract actionable setup errors from captured child-process logs."""
import ast
import json
from pathlib import Path
import re


def text(path):
    return Path(path).read_text(encoding="utf-8", errors="replace") if Path(path).exists() else ""


def install_error(path):
    """Keep the dependency failure, not the wrapper exit or secondary missing file."""
    stderr = text(str(path) + ".err")
    messages = []
    for line in text(path).splitlines():
        match = re.search(r'\bstep=("(?:[^"\\]|\\.)*")$', line)
        if match:
            try:
                line = json.loads(match[1])
            except ValueError:
                # Go's text logger uses \x1b escapes, which are not JSON.
                try:
                    line = ast.literal_eval(match[1])
                except (ValueError, SyntaxError):
                    pass
        line = re.sub(r"(?:\x1b|\\x1b)\[[0-9;]*m", "", line)
        if "ERROR!" in line:
            messages.append(line[line.index("ERROR!") + 6:].strip())
    steam_metadata_error = re.search(r"Info request for AppId \d+ returned error Timeout|Failed to load platform info", text(path))
    if messages and ("SteamCMD could not install" in stderr or steam_metadata_error):
        return "SteamCMD: " + "; ".join(list(dict.fromkeys(messages))[-4:])
    # The first stderr line carries winbed's cause; joined follow-on errors can
    # merely say the probe's destination does not exist after a failed install.
    for line in stderr.splitlines():
        if line.strip():
            return line.removeprefix("winbed: ").split(". If it keeps failing", 1)[0]
    return None


def steam_network_timeout(path):
    return bool(re.search(r"Info request for AppId \d+ returned error Timeout|Failed to load platform info|ERROR! \(Timed out\)|No Connection",
                          text(path), re.IGNORECASE))


def transient_install_error(path):
    stderr = text(str(path) + ".err")
    return "SteamCMD could not install" in stderr and steam_network_timeout(path)
