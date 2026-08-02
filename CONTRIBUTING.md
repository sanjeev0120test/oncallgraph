# Contributing

## Validate changes

GitHub Actions is the source of truth. Prefer push → watch the `ci` workflow.

Optional local smoke (keeps the laptop light; no `-race`):

- Windows: `pwsh scripts/verify.ps1`
- Unix: `bash scripts/verify.sh` or `make quick`

## Fixtures

- Hot incident pack: `fixtures/incident_checkout` (goldens under `expected/`)
- Healthy pack: `fixtures/fleet_healthy`

Update goldens only when behavior intentionally changes:

```bash
go run ./cmd/opsgraph test ./fixtures/incident_checkout --update
```

## Do not commit

- Binaries / release archives (`bin/`, `dist/`, `*.tar.gz`, `*.zip`, `SHA256SUMS`)
- Local planning docs (`PLAN.md`, `AGENTS.md` — gitignored)
- Secrets or `.opsgraph/` runtime data
