---
service: checkout
owner: payments
aliases: [checkout-api, checkout-service]
---

# Checkout runbook

1. A deploy in the last hour is the most likely trigger; inspect it first.
<!-- opsgraph:check=deploy_age_lt:60m -->

2. Confirm checkout has recovered and is healthy before closing out.
<!-- opsgraph:check=service_healthy:checkout -->

3. If not recovered, page the payments on-call and open an incident channel.
<!-- opsgraph:check=manual -->
