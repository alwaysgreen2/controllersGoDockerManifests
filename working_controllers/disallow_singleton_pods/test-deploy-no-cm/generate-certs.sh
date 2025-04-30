#!/bin/bash
set -e

# Generate certs
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout tls.key -out tls.crt -subj "/CN=singleton-pod-blocker.webhook-demo.svc" \
  -addext "subjectAltName=DNS:singleton-pod-blocker.webhook-demo.svc,DNS:singleton-pod-blocker.webhook-demo.svc.cluster.local"

# Create secret
kubectl -n webhook-demo create secret tls webhook-certs \
  --cert=tls.crt \
  --key=tls.key \
  --dry-run=client -o yaml | kubectl apply -f -

# Get CA bundle
CA_BUNDLE=$(base64 -w0 < tls.crt)

# Patch webhook
kubectl patch validatingwebhookconfiguration singleton-pod-blocker \
  --type='json' \
  -p="[{'op':'replace','path':'/webhooks/0/clientConfig/caBundle','value':'${CA_BUNDLE}'}]"

rm tls.crt tls.key
