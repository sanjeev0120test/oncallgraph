---
service: api
owner: platform
aliases: [api-gateway]
---

# API runbook

1. Confirm api is healthy.
<!-- opsgraph:check=service_healthy:api -->

2. Confirm db dependency is healthy.
<!-- opsgraph:check=service_healthy:db -->
