#!/usr/bin/env python3
"""Regenerate the served Structs OpenAPI document from its two RPC services."""
import json
from pathlib import Path
import subprocess
import tempfile


def main():
    root = Path(__file__).resolve().parent.parent
    target = root / "docs/static/openapi.yml"
    previous = json.loads(target.read_text())
    with tempfile.TemporaryDirectory(prefix="structs-openapi-") as directory:
        subprocess.run(["buf", "generate", "--template", "proto/buf.gen.swagger.yaml",
            "--path", "proto/structs/structs/query.proto", "--path", "proto/structs/structs/tx.proto",
            "--output", directory], cwd=root, check=True)
        result = {key: value for key, value in previous.items() if key not in ("paths", "definitions", "tags")}
        # This generator owns the Structs services. Preserve the Cosmos/IBC
        # services published alongside them when refreshing our own schemas.
        result.update(
            paths={path: methods for path, methods in previous["paths"].items()
                if path != "/blockheight" and not path.startswith(("/structs/", "/structs.structs."))},
            definitions={name: schema for name, schema in previous["definitions"].items()
                if not name.startswith("structs.structs.")},
            tags=list(previous.get("tags", [])),
        )
        generated_definitions = {}
        for service in ("query", "tx"):
            generated = json.loads((Path(directory) / f"structs/structs/{service}.swagger.json").read_text())
            for path, methods in generated["paths"].items():
                if path in result["paths"]:
                    raise ValueError(f"duplicate generated path: {path}")
                for method, operation in methods.items():
                    old = previous["paths"].get(path, {}).get(method, {})
                    # Preserve published operation IDs; apply the existing
                    # Structs namespace convention to newly generated methods.
                    if "operationId" in operation:
                        operation["operationId"] = old.get("operationId", "Structs" + operation["operationId"])
                result["paths"][path] = methods
            for name, schema in generated["definitions"].items():
                if name in generated_definitions and generated_definitions[name] != schema:
                    raise ValueError(f"conflicting generated definition: {name}")
                generated_definitions[name] = schema
                # Shared imported schemas already published by the dependency
                # documentation remain authoritative; Structs-owned ones were
                # removed above and are always replaced by generated output.
                result["definitions"].setdefault(name, schema)
            for tag in generated.get("tags", []):
                if tag not in result["tags"]:
                    result["tags"].append(tag)
        target.write_text(json.dumps(result, separators=(",", ":")) + "\n")


if __name__ == "__main__":
    main()
