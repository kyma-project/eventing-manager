# Configure SAP Event Mesh for Kyma Eventing

If you want to use SAP Event Mesh as backend for Kyma Eventing, you must first set up the credentials.

## Prerequisites

- You have the Eventing module and the SAP BTP Operator module in your Kyma cluster (see [Adding and Deleting a Kyma Module](https://kyma-project.io/#/02-get-started/01-quick-install)).
- You are using an enterprise account.

## Procedure

1. Log in to Kyma dashboard. Find the URL in the **Overview** section of your subaccount.
2. To generate an SAP Event Mesh Secret, select a namespace and go to **Service Management** > **Service Instances**.
3. Choose **Create**, and enter the following information:
   - **Name** - enter the name of your instance.
   - **Offering Name** - type: `enterprise-messaging`.
   - **Plan Name** - type: `default`.
4. In **Parameters**, paste the following sample `config.json`:
    ```json
    {
      "options": {
          "management": true,
          "messagingrest": true
      },
      "rules": {
          "topicRules": {
              "publishFilter": [
                  "${namespace}/*"
              ],
              "subscribeFilter": [
                  "${namespace}/*"
              ]
          },
          "queueRules": {
              "publishFilter": [
                  "${namespace}/*"
              ],
              "subscribeFilter": [
                  "${namespace}/*"
              ]
          }
      },
      "resources": {
        "units": "10"
      },
      "version": "1.1.0",
      "emname": "{EVENT_MESH_NAME}",
      "namespace": "{EVENT_MESH_NAMESPACE}"
    }
    ```
5.  Replace `{EVENT_MESH_NAME}` and `{EVENT_MESH_NAMESPACE}` with the values you want your SAP Event Mesh instance to have.
    > **Note:** The `{EVENT_MESH_NAMESPACE}` must follow the [SAP Event Specification](https://help.sap.com/viewer/bf82e6b26456494cbdd197057c09979f/Cloud/en-US/00d56d697c7549408cfacc8cb6a46b11.html); for example, it cannot exceed 64 characters, or begin or end with a dot or hyphen.
6.  Go to **Service Management** > **Service Bindings** and choose **Create**.
7.  Provide the name of your binding, select the name of your instance from the list, and choose **Create**.

## Results

You have set up SAP Event Mesh for Kyma Eventing.

## Next Steps

Choose SAP Event Mesh as your backend for the Eventing module (see [Choose an Eventing Backend](evnt-01-choose-backend.md)).