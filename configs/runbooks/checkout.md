---
service: checkout
owner: payments
aliases: [checkout-api]
---

# Checkout incident runbook

1. Confirm a recent deploy is the likely cause.
<!-- opsgraph:check=deploy_age_lt:60m -->

2. Verify the checkout deployment exists in the cluster.
<!-- opsgraph:check=k8s_deployment_exists:checkout -->

3. Check whether the auth upstream is unhealthy.
<!-- opsgraph:check=service_unhealthy:auth -->

4. Page the payments on-call and open an incident channel.
<!-- opsgraph:check=manual -->
