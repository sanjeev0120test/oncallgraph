# Runbook format

Runbooks are Markdown with optional YAML front matter and HTML-comment check
annotations. `oncallgraph` parses numbered steps and evaluates each check against
the current store.

## Front matter

```yaml
---
service: checkout
owner: payments
aliases: [checkout-api]
---
```

## Checks

Prefer `oncallgraph:check=…` (legacy `opsgraph:check=…` still works). A check
binds to the nearest preceding numbered step.

```markdown
1. Confirm a recent deploy.
<!-- oncallgraph:check=deploy_age_lt:60m -->

2. Service must be healthy.
<!-- oncallgraph:check=service_healthy:checkout -->

3. Human follow-up.
<!-- oncallgraph:check=manual -->
```

### Catalog

| Check | Pass when |
|-------|-----------|
| `deploy_age_lt:Xm` / `deploy_age_gt:Xm` | Newest change age vs window |
| `k8s_deployment_exists:name` | Deployment present in snapshot |
| `service_healthy:name` / `service_unhealthy:name` | Health matches |
| `alert_firing:name` | Alert status is `firing` |
| `manual` | Always manual (never fails the step) |

### Roll-up

- Step: failing deploy/health/k8s/alert checks → `stale`; other failing non-manual → `fail`; parse error → `error`; `manual` → `manual`.
- Runbook: any `fail`/`error` → `fail`; else any `stale` → `stale`; else `pass`; missing runbook → `missing`.
