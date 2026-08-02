# Security Policy

## Supported versions

Security fixes are applied to the latest release on `main` and the most recent `v*` GitHub Release tag.

## Reporting a vulnerability

Please report security issues **privately** via GitHub Security Advisories for this repository:

https://github.com/sanjeev0120test/opsgraph/security/advisories/new

Do not open a public issue for vulnerabilities that could affect users before a fix is available.

Include:

- Affected version / commit
- Reproduction steps (preferably offline / fixture-based)
- Impact assessment

We aim to acknowledge reports within 7 days.

## Repository controls

`main` is protected by a GitHub ruleset that requires the CI aggregator check
`all checks passed` before merges (admins may bypass for emergency fixes).
Direct pushes still run the full Actions matrix afterward; treat a red tip as
not releasable.
