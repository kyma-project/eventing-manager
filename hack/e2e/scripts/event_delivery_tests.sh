#!/usr/bin/env bash
set -e

# for our tests we need to port-forward the eventing-publisher-proxy.
echo "Port-forwarding to the eventing-publisher-proxy using port: 38081"
kubectl -n kyma-system port-forward svc/eventing-publisher-proxy 38081:80 &
PID1=$!

# This will kill all the port-forwarding. We need this to be in a function so we can even call it, if our tests fails
# since `set -e` would stop the script in case of an failing test.
function kill_port_forward() {
  echo "Killing the port-forwarding for port: 38081"
  kill ${PID1}
}
# This kills the port-forwards even if the test fails.
trap kill_port_forward ERR

echo "Running tests..."
go test -v ./hack/e2e/eventing/eventing_test.go --tags=e2e

kill_port_forward
