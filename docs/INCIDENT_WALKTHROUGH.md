# Incident walkthrough

This is the built-in `incident_checkout` scenario, exactly as `oncallgraph demo`
runs it.

## What happened

Around `2026-07-31T11:38:00Z`, checkout was redeployed. Shortly after, error
rate spiked. Upstream `auth` is unhealthy; downstream `order` depends on
checkout.

## Reproduce

```bash
oncallgraph demo
```

You should see: degraded checkout, firing `CheckoutErrorRateHigh`, recent
deploy, unhealthy auth upstream, impacted order downstream, and a runbook that
is `stale` (step 2 expects healthy checkout).

## Dig deeper

```bash
oncallgraph demo --format json | jq '.evidence'
oncallgraph blast checkout --fixture fixtures/incident_checkout
oncallgraph impact auth --fixture fixtures/incident_checkout
oncallgraph why checkout --fixture fixtures/incident_checkout
oncallgraph demo --ai
```
