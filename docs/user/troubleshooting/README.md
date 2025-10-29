# Troubleshooting for the Eventing Module

Troubleshoot problems related to the Eventing module:

- [General Diagnostics: Event Not Delivered](evnt-01-eventing-troubleshooting.md): Start here if your events are not reaching their destination or your Subscription is not Ready.
- [Subscriber Receives Irrelevant Events](evnt-02-subscriber-irrelevant-events.md): Use this guide if a subscriber receives events it did not subscribe to.
- [NATS Backend Storage Is Full](evnt-03-free-jetstream-storage.md): Follow these steps if the Eventing Publisher Proxy returns a 507 Insufficient Storage error.

For issues with the NATS cluster, see the NATS troubleshooting guides:

- [Troubleshooting the NATS Module](https://kyma-project.io/#/nats-manager/user/troubleshooting/README.md): Refer to this guide for issues related to the NATS module itself, such as Pod health or stream configuration.
- [Published Events Are Pending in the Stream](https://github.com/kyma-project/nats-manager/blob/main/docs/user/troubleshooting/03-10-fix-pending-events.md): Use this guide if events are stuck in the NATS stream and are not being delivered.


If you can't find a solution, don't hesitate to create a [GitHub issue](https://github.com/kyma-project/eventing-manager/issues/new/choose).
