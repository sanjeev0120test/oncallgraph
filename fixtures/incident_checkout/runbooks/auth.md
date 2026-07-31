---
service: auth
owner: identity
aliases: [auth-svc, auth-service]
---

# Auth runbook

1. Verify auth is currently unhealthy (this is the upstream cause).
<!-- oncallgraph:check=service_unhealthy:auth -->

2. Escalate to the identity on-call.
<!-- oncallgraph:check=manual -->
