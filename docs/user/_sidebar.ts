export default [
  { text: 'Event Naming and Cleanup', link: './evnt-event-names.md' },
  { text: 'Eventing Metrics', link: './evnt-eventing-metrics.md' },
  { text: 'Choose an Eventing Backend', link: './evnt-01-choose-backend.md' },
  { text: 'Configure SAP Event Mesh for Kyma Eventing', link: './tutorials/evnt-01-configure-event-mesh.md' },
  { text: 'Subscribe to Multiple Event Types', link: './tutorials/evnt-02-subs-with-multiple-filters.md' },
  { text: 'Manage Subscriber Workload with Max-In-Flight', link: './tutorials/evnt-04-change-max-in-flight-in-sub.md' },
  { text: 'Troubleshooting for the Eventing Module', link: './troubleshooting/README.md', collapsed: true, items: [
    { text: 'General Diagnostics: Event Not Delivered', link: './troubleshooting/evnt-01-eventing-troubleshooting.md' },
    { text: 'Subscriber Receives Irrelevant Events', link: './troubleshooting/evnt-02-subscriber-irrelevant-events.md' },
    { text: 'NATS Backend Storage Is Full', link: './troubleshooting/evnt-03-free-jetstream-storage.md' }
    ] },
  { text: 'Resources', link: './resources/README.md', collapsed: true, items: [
    { text: 'Eventing CR', link: './resources/eventing-cr.md' },
    { text: 'Subscription CR', link: './resources/subscription-cr.md' }
    ] }
];
