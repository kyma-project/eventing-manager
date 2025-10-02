# Troubleshooting the NATS Module

## Symptom

- The NATS module is not in a Ready state.
- The Eventing module reports that it cannot connect to the NATS backend.
- NATS Pods are in a `CrashLoopBackOff` or `Pending` state.

## Cause

Issues with the NATS module can stem from misconfigurations in the NATS custom resource, problems with the underlying Kubernetes nodes, or storage issues with Persistent Volume Claims (PVCs).

## Solution

### 1. Check the NATS CR Status

1. To verify the health of the NATS cluster, check the NATS CR:

   ```bash
   kubectl get nats -n kyma-system
   ````

2. Look for `STATE: Ready`. If the state is `Error` or `Processing`, inspect the CR for detailed error messages:

   ```bash
   kubectl get nats {NATS_CR_NAME} -n kyma-system -o yaml
   ```

3. Review the `status.conditions` to understand the root cause.

## 2. Check the NATS Pods

1. Ensure all NATS Pods are running correctly.

   ```bash
   kubectl get pods -n kyma-system -l nats_cluster=eventing-nats
   ```

2. If any Pods are not in the `Running` state, use `kubectl describe pod` and `kubectl logs` to investigate further.

## 3. Check the Persistent Volume Claims (PVCs)

1. If you use file storage, a common issue is a problem with the PVCs.

   ```bash
   kubectl get pvc -n kyma-system -l nats_cluster=eventing-nats
   ```

2. Check the `STATUS` column. If a PVC is `Pending`, there may be no available Persistent Volume that satisfies its request. If it is `Bound`, check if it is full by referring to the "NATS Backend Storage Is Full" guide.

### 4. Inspect the NATS JetStream

If the cluster appears healthy, you can inspect the JetStream components directly using the [NATS CLI](https://github.com/nats-io/natscli).

1. Ensure that you have access to the NATS server (see [Acquiring NATS Server System Account Credentials](https://kyma-project.io/#/nats-manager/user/10-nats-server-system-events)).

1. Port-forward to a NATS Pod:

   ```bash
   kubectl -n kyma-system port-forward svc/eventing-nats 4222
   ```

2. Verify that the `kyma` stream exists.

   ```bash
         $ nats stream ls
     ╭────────────────────────────────────────────────────────────────────────────╮
     │                                  Streams                                   │
     ├──────┬─────────────┬─────────────────────┬──────────┬───────┬──────────────┤
     │ Name │ Description │ Created             │ Messages │ Size  │ Last Message │
     ├──────┼─────────────┼─────────────────────┼──────────┼───────┼──────────────┤
     │ sap  │             │ 2022-05-03 00:00:00 │ 0        │ 318 B │ 5.80s        │
     ╰──────┴─────────────┴─────────────────────┴──────────┴───────┴──────────────╯
   ```

3. If the stream exists, check the timestamp of the `Last Message` that the stream received. A recent timestamp would mean that the event was published correctly.

4. Check if the consumers were created and have the expected configurations.

   ```bash
   nats consumer info
   ```

   To correlate the consumer to the Subscription and the specific event type, check the `description` field of the consumer.

5. If the PVC storage is fully consumed and matches the stream size as shown above, the stream can no longer receive messages. Either increase the PVC storage size (see [NATS Backend Storage Is Full](evnt-03-free-jetstream-storage.md)) or set the `MaxBytes` property which removes the old messages.

