import unittest
import tempfile
from pathlib import Path

from deploy_secrets import (
    DeploySecretsError,
    load_private_deploy_secrets,
    parse_deploy_secrets,
)


class ParseDeploySecretsTests(unittest.TestCase):
    def test_parses_deployment_key_values_with_spaces_and_optional_quotes(self):
        values = parse_deploy_secrets(
            "# deployment values\n"
            "admin_email = operator@example.invalid\n"
            "admin_password = synthetic-password-without-real-account\n"
            'gowa_base_url = "https://gowa.example.invalid/api"\n'
        )

        self.assertEqual(values["admin_email"], "operator@example.invalid")
        self.assertEqual(values["admin_password"], "synthetic-password-without-real-account")
        self.assertEqual(values["gowa_base_url"], "https://gowa.example.invalid/api")

    def test_rejects_duplicate_and_malformed_keys(self):
        with self.assertRaises(DeploySecretsError):
            parse_deploy_secrets("admin_password = first\nadmin_password = second\n")
        with self.assertRaises(DeploySecretsError):
            parse_deploy_secrets("not a key-value entry\n")

    def test_rejects_mismatched_quotes(self):
        with self.assertRaises(DeploySecretsError):
            parse_deploy_secrets('admin_password = "unterminated\n')

    def test_private_loader_rejects_symlinks(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "target.conf"
            target.write_text("admin_password = synthetic-value\n", encoding="utf-8")
            link = Path(directory) / "link.conf"
            link.symlink_to(target)

            with self.assertRaisesRegex(DeploySecretsError, "non-symlink"):
                load_private_deploy_secrets(link)


if __name__ == "__main__":
    unittest.main()
