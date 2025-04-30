#!/bin/bash
set -e

kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: webhook-demo
  labels:
    kubernetes.io/metadata.name: webhook-demo
EOF

./generate-certs.sh

kubectl apply -f webhook.yaml

kubectl wait --for=condition=Available deployment/singleton-pod-blocker -n webhook-demo --timeout=120s

echo "Webhook deployed successfully!"
