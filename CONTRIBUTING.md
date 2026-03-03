# Contributing to Stackyard

Thanks for contributing.

Stackyard grows through staged, test-driven service emulation. Please keep changes incremental, verifiable, and documented.

## Ways to Contribute

- Add or expand an emulated AWS service
- Improve protocol compatibility and error behavior
- Add tests for regressions or edge cases
- Improve examples and documentation
- Improve tooling in `scripts/`

## Development Setup

Prerequisites:

- Go 1.23+
- Docker + Docker Compose
- Python 3 (for coverage tooling)

Common commands:

```bash
make fmt
make tidy
make test
make ci
```

## Preferred Change Pattern

For service work, follow this order:

1. Add/update operation/type catalog
2. Implement candidate/router parsing
3. Implement staged store behavior
4. Add stage tests (`stage0` for catalog/unknown action, then lifecycle stages)
5. Add or update basic/advanced examples
6. Update docs (`docs/index.html`) and coverage script if needed

## Pull Request Expectations

Please include:

- A clear scope statement (service + stage(s) changed)
- Tests that cover new behavior
- Any docs/example updates required by the change
- Notes on intentional compatibility differences

If your PR adds a **new emulated service**, you are expected to also:

- Add or update `examples/aws/<service>/<service>-basic`
- Add or update `examples/aws/<service>/<service>-advanced`
- Add the service to `scripts/awscli-endpoint-coverage.py`
- Verify no endpoints for that service return `NotImplemented` in advanced example and coverage runs

Before opening a PR, run:

```bash
make fmt
make tidy
make test
```

If your change impacts service coverage or examples, also run:

```bash
make examples-docker
make coverage-all
```

## Style and Quality Guidelines

- Keep handlers explicit and deterministic
- Prefer small, focused commits and PRs
- Avoid broad refactors mixed with feature changes
- Preserve existing behavior unless explicitly changing compatibility
- Add tests for every bug fix

## Documentation Guidelines

If behavior changes, update:

- Root README (project-level behavior)
- `docs/index.html` (service-level coverage/usage)
- Example README(s) where relevant

## Community Norms

- Be respectful and specific in reviews
- Assume good intent
- Focus critique on code and behavior, not people
