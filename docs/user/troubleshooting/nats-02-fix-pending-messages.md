# Published Events are Pending in the Stream

## Symptom

You publish events, but some of them are not received by the subscriber and stay pending in the stream.

## Cause

When the NATS `EventingBackend` has more than one replica, and the `Clustering` property on the NATS Server is enabled, a leader is elected for each stream and consumer (see [NATS: JetStream Clustering](https://docs.nats.io/running-a-nats-service/configuration/clustering/jetstream_clustering)).
When the leader is elected, all the messages are replicated across the replicas.

Sometimes replicas can go out of sync with the other replicas. As a result, messages on some consumers can stop being acknowledged and start piling up in the stream.

## Solution

To fix the "broken" consumers with pending messages, trigger a leader reelection. You can do this either on the consumers that have pending messages, or if that fails, on the stream level.

You need the latest version of [NATS CLI](https://github.com/nats-io/natscli) and access to the NATS server (see [Acquiring NATS Server System Account Credentials](https://kyma-project.io/#/nats-manager/user/10-nats-server-system-events)).

### Consumer Leader Reelection

First, find out which consumers have pending messages:

1. Port forward to a NATS replica:

   ```bash
   kubectl port-forward -n kyma-system eventing-nats-0 4222  

2. Run this shell script:

   ```bash
   for consumer in $(nats consumer list -n sap) # sap is the stream name
   do
     nats consumer info sap $consumer -j | jq -c '{name: .name, pending: .num_pending, leader: .cluster.leader}'
   done
   ```

   The output shows each consumer, its pending message count, and its current leader Pod.

   ```bash
   {"name":"ebcabfe5c902612f0ba3ebde7653f30b","pending":25,"leader":"eventing-nats-1"}
   {"name":"c74c20756af53b592f87edebff67bdf8","pending":0,"leader":"eventing-nats-0"}
   ```

3. Note the name of the consumer with a non-zero pending count and the name of its leader (in this example, `ebcabfe5c902612f0ba3ebde7653f30b`).

4. Port-forward to the identified leader Pod:

   ```bash
   kubectl port-forward -n kyma-system eventing-nats-1 4222  
   ```

5. Trigger the leader reelection for that broken consumer:

   ```bash
   nats consumer cluster step-down sap ebcabfe5c902612f0ba3ebde7653f30b
   ```

   After execution, you see a message like the following:

   ```yaml
   New leader elected "eventing-nats-2"
   
   Information for Consumer sap > ebcabfe5c902612f0ba3ebde7653f30b created 2022-10-24T15:49:43+02:00
   ```

6. Check the consumer and confirm that the pending messages start being dispatched.

### Stream Leader Reelection

Sometimes triggering the leader reelection on the broken consumers doesn't work. In that case, you must restart the NATS Pods to trigger leader reelection on the stream level.

1. Command the stream to step down:

   ```bash
   nats stream cluster step-down sap
   ```

2. Check that your result looks like the following example:

   ```bash
   11:08:22 Requesting leader step down of "eventing-nats-1" in a 3 peer RAFT group
   11:08:23 New leader elected "eventing-nats-0"
   
   Information for Stream sap created 2022-10-24 15:47:19
   
                Subjects: kyma.>
                Replicas: 3
                 Storage: File
   ```


### Restart the NATS Pods
If none of the previous steps work, perform a restart of the NATS Pods:

```bash
# assuming we have 3 NATS instances
$ kubectl delete pod -n kyma-system eventing-nats-0 eventing-nats-1 eventing-nats-2 --wait=false
```
