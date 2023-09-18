#!/usr/bin/env bash
set -e

# for our tests we need to port-forward the eventing-publisher-proxy.
echo "Port-forwarding to the svc/eventing-publisher-proxy using port: 38081"
kubectl -n kyma-system port-forward svc/eventing-publisher-proxy 38081:80 &
PID1=$!
echo "Port-forwarding to the svc/test-sink in namespace: eventing-tests using port: 38071"
kubectl -n eventing-tests port-forward svc/test-sink 38071:80 &
PID2=$!

# This will kill all the port-forwarding. We need this to be in a function so we can even call it, if our tests fails
# since `set -e` would stop the script in case of an failing test.
function kill_port_forward() {
  echo "Killing the port-forwarding for port: 38081"
  kill ${PID1}
  echo "Killing the port-forwarding for port: 38071"
  kill ${PID2}
}
# This kills the port-forwards even if the test fails.
trap kill_port_forward ERR

echo "Running tests..."
go test -v ./hack/e2e/eventing/delivery/delivery_test.go --tags=e2e

kill_port_forward
