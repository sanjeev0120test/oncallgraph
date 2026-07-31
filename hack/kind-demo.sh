#!/usr/bin/env bash
# Optional Phase-2 demo: stand up a tiny kind cluster and show how a k8s
# snapshot feeds opsgraph. NOT required for CI (job is if:false) or everyday use.
# Prerequisites: docker, kind, kubectl. Free and local only.
set -euo pipefail

CLUSTER="${OPSGRAPH_KIND_CLUSTER:-opsgraph-demo}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SNAP="$(mktemp -d "${TMPDIR:-/tmp}/opsgraph-snap.XXXXXX")"
trap 'rm -rf "$SNAP"' EXIT

echo "==> create kind cluster (idempotent)"
if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  kind create cluster --name "$CLUSTER"
fi
kubectl config use-context "kind-$CLUSTER" >/dev/null

echo "==> apply demo workload"
kubectl apply -f - <<'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout
  labels:
    app: checkout
spec:
  replicas: 1
  selector:
    matchLabels:
      app: checkout
  template:
    metadata:
      labels:
        app: checkout
    spec:
      containers:
        - name: checkout
          image: nginx:1.27-alpine
          ports:
            - containerPort: 80
YAML

echo "==> export snapshot for opsgraph (plain YAML, no client-go)"
# Capture a minimal snapshot opsgraph can ingest without a live cluster later.
cat >"$SNAP/deployments.yaml" <<YAML
deployments:
  - name: checkout
    namespace: default
    service_id: checkout
    desired: 1
    ready: 1
    updated_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
YAML
cat >"$SNAP/events.yaml" <<'YAML'
events: []
YAML
cat >"$SNAP/releases.yaml" <<YAML
releases:
  - name: checkout
    service_id: checkout
    chart: checkout
    version: 0.1.0
    revision: 1
    updated_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
YAML

CFG="$SNAP/opsgraph.yaml"
cat >"$CFG" <<YAML
version: 1
services:
  checkout:
    owner: demo
owners:
  demo:
    name: Demo Team
connectors:
  git:
    enabled: false
  kubernetes:
    enabled: true
    snapshot: $SNAP
YAML

echo "==> opsgraph ask checkout (from snapshot)"
(cd "$ROOT" && go run ./cmd/opsgraph ask checkout --config "$CFG" --since 60m)

echo "OK - kind demo finished (cluster left running: kind delete cluster --name $CLUSTER)"
