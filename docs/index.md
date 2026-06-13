# Welcome to Gramophile

Gramophile is a stateful manager for your vinyl record collection, providing integration with Discogs, background syncing, collection organization rules, sales management, and advanced wantlist tools.

While Discogs provides an excellent catalog database, it lacks a stateful history, complex physical organization rules (e.g., shelves, spills, density), automated price reductions for sales, and advanced wantlist capabilities. Gramophile bridges this gap by acting as a stateful layer over Discogs.


## Key Features

- **Deep Discogs Integration**: Automated, multi-stage synchronization of collections, wantlists, and marketplace data.
- **Stateful History**: Tracks historical changes to records and wantlists, providing insights beyond what the standard Discogs API offers.
- **Collection Physical Organization (Organisations)**: Support for complex physical space rules, virtual folders, density rules, spills, and placement snapshots.
- **Sales Management**: Track sales, update prices dynamically based on marketplace data, and monitor sales statistics using configurable strategies.
- **Advanced Wantlists**: Custom logic for managing wants and wantlists with historical tracking and state management.
- **Queueing & Eventual Consistency**: Performs all long-running or external API interactions via a background task queue.


## Architecture Overview

Gramophile is built as a set of gRPC and HTTP services:

- **API Server**: Primary interface for user and internal interactions (e.g., authentication, organization management).
- **Background Processing**: A background worker system that processes long-running or scheduled tasks asynchronously.
- **Persistence Layer**: Custom key-value store named `pstore` to persist data as Protobuf messages.
- **Observability**: Instrumented with OpenTelemetry for tracing and Prometheus for performance metrics.

For detailed instructions, refer to the guides:
- [Onboarding Guide](onboarding.md)
- [Syncing Expectations](syncing.md)
- [Organisations Guide](organisations.md)
