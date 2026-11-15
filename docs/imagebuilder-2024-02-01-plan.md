# Azure VM Image Builder (`imagebuilder-2024-02-01`) Staged Plan

## Objective

Emulate the Azure VM Image Builder control plane (`Microsoft.VirtualMachineImages/imageTemplates`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). A run builds and distributes nothing; it records a succeeded run.

## References

Microsoft Learn pages for the `2024-02-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/imagebuilder/virtual-machine-image-templates/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/imagebuilder/virtual-machine-image-templates/get`
- `https://learn.microsoft.com/en-us/rest/api/imagebuilder/virtual-machine-image-templates/run`

## Service Naming

- service key: `imagebuilder`
- display name: `Azure VM Image Builder`
- implementation: `internal/server/provider_azure_imagebuilder.go`
- tests: `internal/server/provider_azure_imagebuilder_test.go`
- example: `examples/azure/imagebuilder-2024-02-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `imageTemplates` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `run` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- templates need a `UserAssigned` identity with exactly one user-assigned identity
- `source.type` must be `PlatformImage`, `ManagedImage` or `SharedImageVersion`
- `distribute` must list at least one target, each with `type` `ManagedImage` (with `imageId`), `SharedImage` (with `galleryImageId`) or `VHD`, and a `runOutputName`
- each `customize` step's `type` must be `Shell`, `PowerShell`, `WindowsRestart`, `WindowsUpdate` or `File`, and each `validate.inVMValidations` step's `Shell`, `PowerShell` or `File`
- `buildTimeoutInMinutes`, when set, must be an integer from 0 to 960
- a new template has no `lastRunStatus`; `run` sets it to `Succeeded`, and clients can't set it

Out of scope for this stage:
- building and distributing images, and the run outputs they produce
- refusing property updates to an existing template (real templates are immutable apart from tags), `cancel` and trigger resources
- long-running operation polling: create and run complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureImageBuilder -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service imagebuilder`
- `python3 scripts/azure-io-contract-coverage.py --service imagebuilder`
- `python3 scripts/azure-doc-contract-coverage.py --service imagebuilder`
