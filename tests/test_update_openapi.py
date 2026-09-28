"""Refreshing Structs schemas must preserve the dependency API surface."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location(
    "update_openapi", Path(__file__).resolve().parents[1] / "proto/update_openapi.py")
GENERATOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GENERATOR)


class OpenAPIRefreshTests(unittest.TestCase):
    def test_preserves_dependency_routes_and_replaces_structs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            target = root / "docs/static/openapi.yml"
            target.parent.mkdir(parents=True)
            dependency = {"get": {"operationId": "BankBalance"}}
            shared = {"description": "current dependency schema"}
            target.write_text(json.dumps({
                "swagger": "2.0", "info": {"title": "Structs"},
                "paths": {
                    "/cosmos/bank/balance": dependency,
                    "/structs/guild": {"get": {"operationId": "PublishedGuild"}},
                    "/structs.structs.Msg/Retired": {},
                },
                "definitions": {"cosmos.Coin": shared, "structs.structs.Retired": {}},
                "tags": [{"name": "Service"}],
            }))

            def generate(command, **_):
                output = Path(command[command.index("--output") + 1]) / "structs/structs"
                output.mkdir(parents=True)
                for service, path in (("query", "/structs/guild"), ("tx", "/structs.structs.Msg/New")):
                    (output / f"{service}.swagger.json").write_text(json.dumps({
                        "paths": {path: {"get": {"operationId": "NewOperation"}}},
                        "definitions": {"structs.structs.Guild": {"type": "object"},
                                        "cosmos.Coin": {"description": "imported schema"}},
                        "tags": [{"name": "Query" if service == "query" else "Msg"}],
                    }))

            with patch.object(GENERATOR, "__file__", str(root / "proto/update_openapi.py")), \
                    patch.object(GENERATOR.subprocess, "run", side_effect=generate):
                GENERATOR.main()
                first = target.read_bytes()
                GENERATOR.main()
                self.assertEqual(target.read_bytes(), first)
            result = json.loads(first)
            self.assertEqual(result["paths"]["/cosmos/bank/balance"], dependency)
            self.assertEqual(result["definitions"]["cosmos.Coin"], shared)
            self.assertNotIn("/structs.structs.Msg/Retired", result["paths"])
            self.assertNotIn("structs.structs.Retired", result["definitions"])
            self.assertEqual(result["paths"]["/structs/guild"]["get"]["operationId"], "PublishedGuild")
            self.assertEqual(result["paths"]["/structs.structs.Msg/New"]["get"]["operationId"], "StructsNewOperation")
            self.assertIn({"name": "Service"}, result["tags"])


if __name__ == "__main__":
    unittest.main()
