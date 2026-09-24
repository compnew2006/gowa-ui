import tempfile
import unittest
from pathlib import Path

from gowa_device_map import (
    DeviceMapError,
    load_private_mappings,
    parse_mappings,
    render_psql_copy,
)


class ParseMappingsTests(unittest.TestCase):
    def test_parses_tab_separated_device_fixes(self):
        mappings = parse_mappings(
            "# old id<TAB>full id<TAB>jid<TAB>organization_id\n"
            "synthetic-old-device-001\tsynthetic-full-device-001\t"
            "test-user-001@invalid.example\t00000000-0000-4000-8000-000000000001\n"
        )

        self.assertEqual(len(mappings), 1)
        self.assertEqual(mappings[0].old_device_id, "synthetic-old-device-001")
        self.assertEqual(mappings[0].full_device_id, "synthetic-full-device-001")
        self.assertEqual(mappings[0].whatsapp_jid, "test-user-001@invalid.example")
        self.assertEqual(
            mappings[0].organization_id,
            "00000000-0000-4000-8000-000000000001",
        )

    def test_rejects_rows_with_missing_columns(self):
        with self.assertRaises(DeviceMapError):
            parse_mappings("synthetic-old\tsynthetic-full\ttest-user@invalid.example\n")

    def test_rejects_duplicate_old_or_full_ids(self):
        with self.assertRaises(DeviceMapError):
            parse_mappings(
                "synthetic-old\tsynthetic-new-1\ttest-1@invalid.example\t"
                "00000000-0000-4000-8000-000000000001\n"
                "synthetic-old\tsynthetic-new-2\ttest-2@invalid.example\t"
                "00000000-0000-4000-8000-000000000002\n"
            )
        with self.assertRaises(DeviceMapError):
            parse_mappings(
                "synthetic-old-1\tsynthetic-new\ttest-1@invalid.example\t"
                "00000000-0000-4000-8000-000000000001\n"
                "synthetic-old-2\tsynthetic-new\ttest-2@invalid.example\t"
                "00000000-0000-4000-8000-000000000002\n"
            )

    def test_rejects_blank_fields_and_embedded_newlines(self):
        with self.assertRaises(DeviceMapError):
            parse_mappings(
                "synthetic-old\t\ttest-user@invalid.example\t"
                "00000000-0000-4000-8000-000000000001\n"
            )
        with self.assertRaises(DeviceMapError):
            parse_mappings(
                'synthetic-old\t"synthetic\nname"\ttest-user@invalid.example\t'
                "00000000-0000-4000-8000-000000000001\n"
            )
        with self.assertRaises(DeviceMapError):
            parse_mappings(
                "synthetic-old\tsynthetic-full\ttest-user@invalid.example\tinvalid-org\n"
            )

    def test_rendered_copy_stream_updates_only_matching_rows(self):
        mappings = parse_mappings(
            "synthetic-old-device-001\tsynthetic-full-device-001\t"
            "test-user-001@invalid.example\t00000000-0000-4000-8000-000000000001\n"
        )

        output = render_psql_copy(mappings, "https://gowa.example.invalid")

        self.assertIn("\\copy device_map", output)
        self.assertIn(
            "synthetic-old-device-001\tsynthetic-full-device-001\t"
            "test-user-001@invalid.example\t00000000-0000-4000-8000-000000000001\t"
            "https://gowa.example.invalid",
            output,
        )
        self.assertIn("LOCK TABLE whatsapp_accounts IN SHARE ROW EXCLUSIVE MODE", output)
        self.assertIn("device map must match exactly one account per row", output)
        self.assertIn("account.organization_id = mapping.organization_id", output)
        self.assertIn("rtrim(account.gowa_base_url, '/') = mapping.gowa_base_url", output)
        self.assertIn("device map target collides with an existing account", output)
        self.assertIn("WHERE account.gowa_device_id = mapping.old_device_id", output)
        self.assertIn("WHERE account.gowa_device_id IS NOT NULL", output)
        self.assertIn("rtrim(account.gowa_base_url, '/') = (SELECT min(gowa_base_url) FROM device_map)", output)
        self.assertIn("account.organization_id IN (SELECT organization_id FROM device_map)", output)
        self.assertLess(output.index("COMMIT;"), output.index("SELECT 'MAPPING_COMMITTED'"))
        self.assertIn("\\.\n", output)
        self.assertLess(output.index("RAISE EXCEPTION"), output.index("UPDATE whatsapp_accounts"))

    def test_render_rejects_unsafe_gowa_base_url(self):
        mappings = parse_mappings(
            "synthetic-old\tsynthetic-full\ttest-user@invalid.example\t"
            "00000000-0000-4000-8000-000000000001\n"
        )

        for base_url in ["https://user:password@gowa.example.invalid", "https://gowa.example.invalid\n"]:
            with self.subTest(base_url="<redacted>"):
                with self.assertRaises(DeviceMapError):
                    render_psql_copy(mappings, base_url)

    def test_private_map_loader_rejects_symlinks(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "target.tsv"
            target.write_text(
                "synthetic-old\tsynthetic-full\ttest-user@invalid.example\t"
                "00000000-0000-4000-8000-000000000001\n",
                encoding="utf-8",
            )
            link = Path(directory) / "link.tsv"
            link.symlink_to(target)

            with self.assertRaisesRegex(DeviceMapError, "non-symlink"):
                load_private_mappings(link)


if __name__ == "__main__":
    unittest.main()
