#!/usr/bin/env python3
"""Write a validation-only copy of a botocore service model.

Keeps shapes (type, required, members/member/key/value, min, max, pattern,
enum) and operation input/output names; drops docs. Usage:
  gen-aws-model-slim.py [--wire-names] <service-2.json> <out.json>
--wire-names keys body members by their JSON wire name (locationName), for
REST-JSON models whose members are PascalCase but travel camelCase.
The Pipes copy came from the AWS CLI 2.41 model:
  scripts/gen-aws-model-slim.py .../botocore/data/pipes/2015-10-07/service-2.json internal/awsmodels/pipes/model.json
"""
import json
import sys

KEEP = ("type", "required", "min", "max", "pattern", "enum", "sensitive", "document")
MEMBER_KEEP = ("shape", "location", "locationName")

args = sys.argv[1:]
wire = "--wire-names" in args
src, out = [a for a in args if a != "--wire-names"]
with open(src) as fh:
    model = json.load(fh)
shapes = {}
for name, shape in model["shapes"].items():
    slim = {k: shape[k] for k in KEEP if k in shape}
    if "members" in shape:
        slim["members"] = {m: {k: v[k] for k in MEMBER_KEEP if k in v} for m, v in shape["members"].items()}
    for ref in ("member", "key", "value"):
        if ref in shape:
            slim[ref] = {"shape": shape[ref]["shape"]}
    if wire and "members" in slim:
        names = {m: v["locationName"] for m, v in slim["members"].items() if "locationName" in v and "location" not in v}
        slim["members"] = {names.get(m, m): v for m, v in slim["members"].items()}
        if "required" in slim:
            slim["required"] = [names.get(m, m) for m in slim["required"]]
    shapes[name] = slim
ops = {
    name: {"input": op.get("input", {}).get("shape", ""), "output": op.get("output", {}).get("shape", "")}
    for name, op in model["operations"].items()
}
with open(out, "w") as fh:
    json.dump({"operations": ops, "shapes": shapes}, fh, separators=(",", ":"), sort_keys=True)
    fh.write("\n")
