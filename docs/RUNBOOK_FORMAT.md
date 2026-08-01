# Runbook format

Runbooks are Markdown with optional YAML front matter and HTML-comment check
annotations. `opsgraph` parses numbered steps and evaluates each check against
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

Annotate steps with `opsgraph:check=…`. A check
binds to the nearest preceding numbered step.

```markdown
1. Confirm a recent deploy.
<!-- opsgraph:check=deploy_age_lt:60m -->

2. Service must be healthy.
<!-- opsgraph:check=service_healthy:checkout -->

3. Human follow-up.
<!-- opsgraph:check=manual -->
```

### Catalog

| Check | Pass when |
|-------|-----------|
| `deploy_age_lt:Xm` / `deploy_age_gt:Xm` | Newest **deploy/rollout** age vs window (commits ignored) |
| `k8s_deployment_exists:name` | Rollout evidence `ev-k8s-rollout-<name>` present, or service has kubernetes source |
| `service_healthy:name` / `service_unhealthy:name` | Health matches |
| `alert_firing:name` | Alert status is `firing` or `pending` (active) |
| `manual` | Always manual (never fails the step) |

### Roll-up

- Step: failing deploy/health/k8s/alert checks → `stale`; other failing non-manual → `fail`; parse error → `error`; `manual` → `manual`.
- Runbook: any `fail`/`error` → `fail`; else any `stale` → `stale`; else `pass`; missing runbook → `missing`.
