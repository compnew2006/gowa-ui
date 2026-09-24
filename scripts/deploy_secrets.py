#!/usr/bin/env python3
"""Read a single value from the root-only gowa-ui deployment secrets file."""

from __future__ import annotations

import argparse
import os
import re
import stat
import sys
from pathlib import Path


class DeploySecretsError(ValueError):
    """Raised when deployment secrets are unsafe or malformed."""


def parse_deploy_secrets(contents: str) -> dict[str, str]:
    values: dict[str, str] = {}
    for line_number, raw_line in enumerate(contents.splitlines(), start=1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            raise DeploySecretsError(f"line {line_number} is not a key-value entry")

        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", key):
            raise DeploySecretsError(f"line {line_number} has an invalid key")
        if key in values:
            raise DeploySecretsError(f"line {line_number} duplicates a key")
        if value and (value.startswith(('"', "'")) or value.endswith(('"', "'"))):
            if len(value) < 2 or value[0] != value[-1] or value[0] not in ('"', "'"):
                raise DeploySecretsError(f"line {line_number} has mismatched quotes")
            value = value[1:-1]
        if any(ord(char) < 32 or ord(char) == 127 for char in value):
            raise DeploySecretsError(f"line {line_number} contains a control character")
        values[key] = value
    return values


def load_private_deploy_secrets(path: Path) -> dict[str, str]:
    if path.is_symlink() or not path.is_file():
        raise DeploySecretsError("deployment secrets must be a regular, non-symlink file")

    descriptor = None
    try:
        descriptor = os.open(path, os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
        file_stat = os.fstat(descriptor)
        if not stat.S_ISREG(file_stat.st_mode):
            raise DeploySecretsError("deployment secrets must be a regular file")
        if file_stat.st_uid != 0:
            raise DeploySecretsError("deployment secrets must be owned by root")
        if stat.S_IMODE(file_stat.st_mode) & 0o077:
            raise DeploySecretsError("deployment secrets permissions must be 0600 or stricter")

        with os.fdopen(descriptor, "r", encoding="utf-8", newline="") as stream:
            descriptor = None
            return parse_deploy_secrets(stream.read())
    except OSError as exc:
        raise DeploySecretsError("deployment secrets could not be read safely") from exc
    except UnicodeError as exc:
        raise DeploySecretsError("deployment secrets must be UTF-8") from exc
    finally:
        if descriptor is not None:
            os.close(descriptor)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("secrets_file", type=Path)
    parser.add_argument("key")
    args = parser.parse_args()

    try:
        values = load_private_deploy_secrets(args.secrets_file)
    except DeploySecretsError as exc:
        print(f"deployment secrets rejected: {exc}", file=sys.stderr)
        return 1

    print(values.get(args.key, ""), end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
