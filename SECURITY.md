# Security Policy

## Scope

Stackyard is a local development emulator and is not intended to be deployed as a production AWS replacement.

Security issues are still important, especially when they affect:

- unintended remote exposure
- credential handling/signing logic
- request validation bypasses
- denial-of-service conditions in default local usage

## Supported Versions

Security fixes are generally applied to the latest state of the default branch.

## Reporting a Vulnerability

Please do not open public issues for suspected vulnerabilities.

Instead, report privately by contacting the maintainer/repository owner through GitHub security reporting channels (or direct private contact if provided by the project owner).

When reporting, include:

- affected commit/branch
- reproduction steps
- expected vs actual behavior
- impact assessment
- proof-of-concept (if available)

## Response Process

The project aims to:

1. Acknowledge receipt promptly
2. Reproduce and triage severity
3. Prepare and validate a fix
4. Publish the fix and advisory notes as appropriate

## Disclosure

Please allow reasonable time for triage and remediation before public disclosure.

Coordinated disclosure is preferred.

## Hardening Guidance for Users

Even for local use:

- Do not expose Stackyard to untrusted networks
- Use non-production credentials only
- Keep dependencies up to date
- Run tests/CI after pulling updates
