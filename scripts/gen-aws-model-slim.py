#!/usr/bin/env python3
"""Write a validation-only copy of a botocore service model.

Keeps shapes (type, required, members/member/key/value, min, max, pattern,
enum) and operation input/output names; drops docs. Usage:
  gen-aws-model-slim.py <service-2.json> <out.json>
The Pipes copy came from the AWS CLI 2.41 model:
  scripts/gen-aws-model-slim.py .../botocore/data/pipes/2015-10-07/service-2.json internal/awsmodels/pipes/model.json
"""
import json
import sys

KEEP = ("type", "required", "min", "max", "pattern", "enum", "sensitive")
MEMBER_KEEP = ("shape", "location", "locationName")

src, out = sys.argv[1], sys.argv[2]
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
    shapes[name] = slim
ops = {
    name: {"input": op.get("input", {}).get("shape", ""), "output": op.get("output", {}).get("shape", "")}
    for name, op in model["operations"].items()
}
with open(out, "w") as fh:
    json.dump({"operations": ops, "shapes": shapes}, fh, separators=(",", ":"), sort_keys=True)
    fh.write("\n")
