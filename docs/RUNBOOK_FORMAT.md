# Runbook format

Runbooks are plain Markdown with optional YAML front matter and inline check
annotations. `opsgraph` parses numbered steps and evaluates each check against
current incident state.

## Front matter

```markdown
---
service: checkout
owner: payments
aliases: [checkout-api, checkout-service]
---
```

- `service` (required): the service id this runbook belongs to.
- `owner` (optional): owner id.
- `aliases` (optional): alternate names.

## Steps and checks

A step is a Markdown numbered list item. An optional check is an HTML comment
placed on a following line and binds to the nearest preceding step:

```markdown
1. A deploy in the last hour is the most likely trigger; inspect it first.
<!-- opsgraph:check=deploy_age_lt:60m -->

2. Confirm checkout has recovered and is healthy before closing out.
<!-- opsgraph:check=service_healthy:checkout -->

3. Page the payments on-call.
<!-- opsgraph:check=manual -->
```

A step with no annotation is treated as `manual`.

## Supported checks

| Check                              | Passes when...                                        |
|------------------------------------|-------------------------------------------------------|
| `deploy_age_lt:<dur>`              | the latest change/deploy is younger than `<dur>`      |
| `deploy_age_gt:<dur>`              | the latest change/deploy is older than `<dur>`        |
| `k8s_deployment_exists:<name>`     | a deployment for `<name>` is in the snapshot          |
| `service_healthy:<name>`           | `<name>` health is `healthy`                          |
| `service_unhealthy:<name>`         | `<name>` health is `degraded` or `unhealthy`          |
| `alert_firing:<name>`              | an alert named `<name>` (or on service `<name>`) fires|
| `manual`                           | never automated; always reported as `manual`          |

Durations use Go syntax: `30m`, `1h`, `90m`, etc.

## Step and rollup statuses

Each step resolves to one of: `pass`, `stale`, `manual`, `error`.
- A **state mismatch** (e.g. `service_healthy` when the service is degraded)
  reports `stale` - the runbook's assumption no longer holds.
- An **unknown check** or bad argument reports `error`.

The runbook rollup is:
- `fail` if any step is `error`,
- else `stale` if any step is `stale`,
- else `pass`.

`missing` is reported when a service has no runbook.
