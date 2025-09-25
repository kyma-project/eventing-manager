# Choose an Eventing Backend

The eventing module supports two backends: NATS and SAP Event Mesh. Learn how to set up your preferred eventing backend.

## Prerequisites

- You have the Eventing module in your Kyma cluster (see [Adding and Deleting a Kyma Module](https://kyma-project.io/#/02-get-started/01-quick-install)).
- If you want to use NATS, you have added the NATS module.
- If you want to use SAP Event Mesh, you have configured it for Kyma Eventing (see [Configure SAP Event Mesh for Kyma Eventing](evnt-01-configure-event-mesh.md)).

## Context

You can send and receive events in Kyma with Eventing module. By default, Kyma clusters have no eventing backend defined. You must set up an eventing backend, either based on the NATS technology or SAP Event Mesh.

## Procedure

1. Log in to Kyma dashboard. The URL is in the **Overview** section of your subaccount.
2. Go to the `kyma-system` namespace.
3. Go to **Kyma** > **Eventing**, open the `eventing` resource, and choose **Edit**.
4. Under **Backend Type**, select your preferred backend.
5. If you selected **EventMesh**, go to **Event Mesh Secret**, and select the namespace and name of your SAP Event Mesh service binding.
   > **Remember:** If you selected **NATS**, make sure the NATS module is added.
6. Choose **Save**.

## Results

You have set up your eventing backend for Kyma.

If you want to choose another backend, edit the `Eventing` resource again, select the preferred backend, and save your changes.

You can no longer return to an empty eventing backend.