#!/usr/bin/env python3
"""Validate a private device-fix TSV and stream a parameter-safe psql update."""

from __future__ import annotations

import argparse
import csv
import io
import os
import stat
import sys
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import urlsplit
from uuid import UUID


class DeviceMapError(ValueError):
    """Raised when a device mapping file is unsafe or malformed."""


@dataclass(frozen=True)
class DeviceMapping:
    old_device_id: str
    full_device_id: str
    whatsapp_jid: str
    organization_id: str


def parse_mappings(contents: str) -> list[DeviceMapping]:
    mappings: list[DeviceMapping] = []
    seen_old: set[str] = set()
    seen_full: set[str] = set()

    try:
        rows = csv.reader(io.StringIO(contents, newline=""), delimiter="\t", strict=True)
        for line_number, row in enumerate(rows, start=1):
            if not row or (len(row) == 1 and not row[0].strip()):
                continue
            if row[0].lstrip().startswith("#"):
                continue
            if len(row) != 4:
                raise DeviceMapError(
                    f"line {line_number} must contain old_device_id, full_device_id, jid, and organization_id"
                )
            if any(not value or value != value.strip() for value in row):
                raise DeviceMapError(f"line {line_number} contains a blank or padded field")
            if any(any(ord(char) < 32 or ord(char) == 127 for char in value) for value in row):
                raise DeviceMapError(f"line {line_number} contains a control character")

            old_id, full_id, jid, organization_id = row
            try:
                organization_id = str(UUID(organization_id))
            except ValueError as exc:
                raise DeviceMapError(f"line {line_number} has an invalid organization_id") from exc
            if old_id in seen_old:
                raise DeviceMapError(f"line {line_number} duplicates an old device id")
            if full_id in seen_full:
                raise DeviceMapError(f"line {line_number} duplicates a full device id")
            seen_old.add(old_id)
            seen_full.add(full_id)
            mappings.append(DeviceMapping(old_id, full_id, jid, organization_id))
    except csv.Error as exc:
        raise DeviceMapError("mapping file is not valid tab-separated CSV") from exc

    if not mappings:
        raise DeviceMapError("mapping file contains no device mappings")
    return mappings


def load_private_mappings(path: Path) -> list[DeviceMapping]:
    if path.is_symlink() or not path.is_file():
        raise DeviceMapError("mapping file must be a regular, non-symlink file")

    descriptor: int | None = None
    try:
        flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
        descriptor = os.open(path, flags)
        file_stat = os.fstat(descriptor)
        if not stat.S_ISREG(file_stat.st_mode):
            raise DeviceMapError("mapping file must be a regular file")
        if file_stat.st_uid != 0:
            raise DeviceMapError("mapping file must be owned by root")
        if stat.S_IMODE(file_stat.st_mode) & 0o077:
            raise DeviceMapError("mapping file permissions must be 0600 or stricter")

        with os.fdopen(descriptor, "r", encoding="utf-8", newline="") as stream:
            descriptor = None
            return parse_mappings(stream.read())
    except OSError as exc:
        raise DeviceMapError("mapping file could not be read") from exc
    except UnicodeError as exc:
        raise DeviceMapError("mapping file must be UTF-8") from exc
    finally:
        if descriptor is not None:
            os.close(descriptor)


def render_psql_copy(mappings: list[DeviceMapping], gowa_base_url: str) -> str:
    if not mappings:
        raise DeviceMapError("mapping file contains no device mappings")
    if (
        not gowa_base_url
        or gowa_base_url != gowa_base_url.strip()
        or any(ord(char) <= 32 or ord(char) == 127 for char in gowa_base_url)
    ):
        raise DeviceMapError("GOWA base URL is required and must not contain whitespace or control characters")
    try:
        parsed_base = urlsplit(gowa_base_url)
    except ValueError as exc:
        raise DeviceMapError("GOWA base URL must be a valid HTTP(S) URL") from exc
    if (
        parsed_base.scheme not in ("http", "https")
        or not parsed_base.hostname
        or parsed_base.username
        or parsed_base.password
        or parsed_base.query
        or parsed_base.fragment
    ):
        raise DeviceMapError("GOWA base URL must not contain embedded credentials, query data, or a fragment")
    normalized_base_url = gowa_base_url.rstrip("/")

    output = io.StringIO()
    output.write("BEGIN;\n")
    output.write(
        "CREATE TEMP TABLE device_map ("
        "old_device_id text PRIMARY KEY, "
        "full_device_id text NOT NULL UNIQUE, "
        "whatsapp_jid text NOT NULL, "
        "organization_id uuid NOT NULL, "
        "gowa_base_url text NOT NULL) ON COMMIT PRESERVE ROWS;\n"
    )
    output.write(
        r"\copy device_map (old_device_id, full_device_id, whatsapp_jid, organization_id, gowa_base_url) "
        r"FROM STDIN WITH (FORMAT csv, DELIMITER E'\t')"
        "\n"
    )
    writer = csv.writer(output, delimiter="\t", quoting=csv.QUOTE_MINIMAL, lineterminator="\n")
    writer.writerows(
        (
            mapping.old_device_id,
            mapping.full_device_id,
            mapping.whatsapp_jid,
            mapping.organization_id,
            normalized_base_url,
        )
        for mapping in mappings
    )
    output.write("\\.\n")
    output.write(
        "LOCK TABLE whatsapp_accounts IN SHARE ROW EXCLUSIVE MODE;\n"
        "DO $$\n"
        "BEGIN\n"
        "  IF EXISTS (\n"
        "    SELECT 1 FROM device_map AS mapping\n"
        "    WHERE (SELECT count(*) FROM whatsapp_accounts AS account\n"
        "           WHERE account.gowa_device_id = mapping.old_device_id\n"
        "             AND account.organization_id = mapping.organization_id\n"
        "             AND rtrim(account.gowa_base_url, '/') = mapping.gowa_base_url) <> 1\n"
        "  ) THEN\n"
        "    RAISE EXCEPTION 'device map must match exactly one account per row';\n"
        "  END IF;\n"
        "  IF EXISTS (\n"
        "    SELECT 1 FROM device_map AS target_mapping\n"
        "    JOIN whatsapp_accounts AS existing\n"
        "      ON existing.gowa_device_id = target_mapping.full_device_id\n"
        "     AND rtrim(existing.gowa_base_url, '/') = target_mapping.gowa_base_url\n"
        "    WHERE NOT EXISTS (\n"
        "      SELECT 1 FROM device_map AS source_mapping\n"
        "      WHERE source_mapping.old_device_id = existing.gowa_device_id\n"
        "        AND source_mapping.organization_id = existing.organization_id\n"
        "        AND source_mapping.gowa_base_url = target_mapping.gowa_base_url\n"
        "    )\n"
        "  ) THEN\n"
        "    RAISE EXCEPTION 'device map target collides with an existing account';\n"
        "  END IF;\n"
        "END $$;\n"
        "UPDATE whatsapp_accounts AS account\n"
        "SET gowa_device_id = mapping.full_device_id,\n"
        "    gowa_jid = mapping.whatsapp_jid\n"
        "FROM device_map AS mapping\n"
        "WHERE account.gowa_device_id = mapping.old_device_id\n"
        "  AND account.organization_id = mapping.organization_id\n"
        "  AND rtrim(account.gowa_base_url, '/') = mapping.gowa_base_url;\n"
        "COMMIT;\n"
        "SELECT 'MAPPING_COMMITTED';\n"
        "SELECT 'DEVICE:' || account.gowa_device_id\n"
        "FROM whatsapp_accounts AS account\n"
        "WHERE account.gowa_device_id IS NOT NULL\n"
        "  AND rtrim(account.gowa_base_url, '/') = (SELECT min(gowa_base_url) FROM device_map)\n"
        "  AND account.organization_id IN (SELECT organization_id FROM device_map)\n"
        "ORDER BY account.gowa_device_id;\n"
        "DROP TABLE device_map;\n"
    )
    return output.getvalue()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mapping_file", type=Path)
    parser.add_argument("gowa_base_url")
    args = parser.parse_args()

    try:
        mappings = load_private_mappings(args.mapping_file)
        sys.stdout.write(render_psql_copy(mappings, args.gowa_base_url))
    except DeviceMapError as exc:
        print(f"device map rejected: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
