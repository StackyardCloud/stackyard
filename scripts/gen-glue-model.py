#!/usr/bin/env python3
"""Regenerate internal/awsmodels/glue/model.json from the installed botocore.

The emulator uses it to validate request shapes and to shape responses so
they always match the model. Usage:
    python3 scripts/gen-glue-model.py [botocore-data-dir]
Pass the AWS CLI v2 bundled data dir (awscli/botocore/data) to match the CLI.
"""
import json
import sys
from pathlib import Path

import botocore.session

OUT = Path(__file__).resolve().parent.parent / "internal/awsmodels/glue/model.json"


def main() -> None:
    session = botocore.session.get_session()
    if len(sys.argv) > 1:
        session.get_component("data_loader").search_paths.insert(0, sys.argv[1])
    model = session.get_service_model("glue")
    shapes: dict[str, dict] = {}

    def visit(shape) -> str:
        name = shape.name
        if name in shapes:
            return name
        spec: dict = {"t": shape.type_name}
        shapes[name] = spec
        meta = shape.metadata
        for key in ("min", "max"):
            if key in meta:
                spec[key] = meta[key]
        if getattr(shape, "enum", None):
            spec["e"] = list(shape.enum)
        if shape.type_name == "structure":
            spec["m"] = {k: visit(v) for k, v in shape.members.items()}
            if shape.required_members:
                spec["r"] = list(shape.required_members)
            if getattr(shape, "is_document_type", False):
                spec["doc"] = True
        elif shape.type_name == "list":
            spec["mem"] = visit(shape.member)
        elif shape.type_name == "map":
            spec["k"] = visit(shape.key)
            spec["v"] = visit(shape.value)
        return name

    ops = {}
    for op_name in model.operation_names:
        op = model.operation_model(op_name)
        ops[op_name] = {
            "in": visit(op.input_shape) if op.input_shape is not None else "",
            "out": visit(op.output_shape) if op.output_shape is not None else "",
        }
    OUT.write_text(json.dumps({"version": model.api_version, "operations": ops, "shapes": shapes}, sort_keys=True, separators=(",", ":")) + "\n")
    print(f"wrote {OUT} ({len(ops)} operations, {len(shapes)} shapes)")


if __name__ == "__main__":
    main()
