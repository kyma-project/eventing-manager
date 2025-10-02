# Troubleshooting for the Eventing Module

Troubleshoot problems related to the Eventing module:

- [General Diagnostics: Event Not Delivered](evnt-01-eventing-troubleshooting.md)
  Start here if your events are not reaching their destination or your Subscription is not Ready.
- [Subscriber Receives Irrelevant Events]()
  Use this guide if a subscriber receives events it did not subscribe to.
- [NATS Backend Storage Is Full]()
  Follow these steps if the Eventing Publisher Proxy returns a 507 Insufficient Storage error.

For issues with the NATS cluster, see the NATS troubleshooting guides:
 <!-- replace these links after moving the docs to NATS repo -->
- [Troubleshooting the NATS Module](nats-01-module-troubleshooting.md)
  Refer to this guide for issues related to the NATS module itself, such as Pod health or stream configuration.
- [Events Are Pending in the NATS Stream](nats-02-fix-pending-messages.md)
  Use this guide if events are stuck in the NATS stream and are not being delivered.


If you can't find a solution, don't hesitate to create a [GitHub issue](https://github.com/kyma-project/kyma/issues).
