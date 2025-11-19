# Subscriber Receives Irrelevant Events

## Symptom

A subscriber receives events that don't match the types defined in its Subscription custom resource.

## Cause

This issue occurs due to a naming collision after event type cleanup. To conform to Cloud Event specifications, Eventing modifies the event names to filter out prohibited characters. For details, see [Event name cleanup](../evnt-event-names.md#event-name-cleanup).

## Solution

1. To find the cleaned event type that the backend uses, check the status of the Subscription:

   ```bash
   kubectl get subscription {SUBSCRIPTION_NAME} -n {NAMESPACE} -o jsonpath='{.status.types}'
   ```

   This command returns the mapping between the original type and the cleaned type. Note the **cleanType** value.

2. Search all Subscription resources in the cluster to see if another one uses the same cleaned type. Replace **{CLEAN_TYPE}** with the value from the previous step.

   ```bash
   kubectl get subscriptions -A -o yaml | grep "cleanType: {CLEAN_TYPE}" -B 1
   ```

   If the output shows more than one Subscription with the same **cleanType**, you have a naming collision.

3. To fix the issue, modify the **spec.types** in one of the colliding Subscription resources and update the corresponding publisher to use a unique event type that doesn't cause a collision.
