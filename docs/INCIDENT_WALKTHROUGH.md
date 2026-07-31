# Incident walkthrough

This is the built-in `incident_checkout` scenario, exactly as `opsgraph demo`
prints it. It shows how the tool turns scattered signals into one answer.

## The situation

You're on-call for payments. PagerDuty fires: checkout error rate is high.

## One command

```bash
opsgraph demo
```

```
SERVICE   checkout (degraded)   owner: Payments Team <payments@example.com>
WINDOW    last 60m (as of 2026-07-31T12:00:00Z)
CHANGES   deploy  22m ago  abc123  "deploy checkout v1.4.2"  [ev-change-1]
          rollout  22m ago  k8s-rollout-checkout  "rollout checkout (1/3 ready)"  [ev-k8s-rollout-checkout]
ALERTS    CheckoutErrorRateHigh (critical, firing)  [ev-alert-1]
BLAST     upstream: auth (unhealthy), redis (unknown)   downstream: order (healthy)
RUNBOOK   runbooks/checkout.md -> STALE (1 pass, 1 stale, 1 manual)
          1. [pass  ] A deploy in the last hour is the most likely trigger; inspect it first.
          2. [stale ] Confirm checkout has recovered and is healthy before closing out.
          3. [manual] If not recovered, page the payments on-call and open an incident channel.
NEXT
          1. Inspect the most recent deploy first: "deploy checkout v1.4.2" (abc123).
          2. Check upstream auth (unhealthy) before changing checkout.
          3. Acknowledge firing alert CheckoutErrorRateHigh and correlate it with the recent change.
          4. Runbook runbooks/checkout.md is stale - review step(s) 2.
          5. Notify owner Payments Team <payments@example.com>.
```

## How to read it

- **SERVICE** - checkout is `degraded` (1/3 replicas ready in the snapshot). Owner is known.
- **CHANGES** - a deploy 22m ago (`abc123`) plus the matching rollout. Prime suspect.
- **ALERTS** - the firing alert, with an evidence id.
- **BLAST** - `auth` upstream is unhealthy (likely the real cause); `redis` is a
  dependency with no service row, so it's synthesized as `unknown`. `order` is
  downstream and may be impacted.
- **RUNBOOK** - the runbook is `stale`: step 2 assumes checkout is healthy, but it
  isn't. That assumption needs updating.
- **NEXT** - deterministic, ordered next steps. Every claim above is backed by an
  `[ev-*]` evidence id.

## Evidence and JSON

Every fact carries an evidence id. For machine use:

```bash
opsgraph demo --format json | jq '.evidence'
```

## Optional AI

```bash
opsgraph demo --ai
```

Adds a short natural-language summary. With a local Ollama model it uses RAG over
the incident's evidence; without it, it falls back to a deterministic offline
summary - so it always works and is always free.
