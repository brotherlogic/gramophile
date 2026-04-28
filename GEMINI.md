# Gramophile Project Overview

Gramophile is a sophisticated Go-based backend system designed to enhance the experience of managing a vinyl record collection integrated with Discogs. It provides a stateful layer that tracks collection history, handles complex background tasks, and implements a robust queuing system for asynchronous operations.

## Core Architecture

The system is built as a set of gRPC and HTTP services supported by a custom persistence layer and a background worker system.

-   **API Server**: Serves as the primary interface for user and internal interactions. It handles requests ranging from authentication to complex collection organization.
-   **Background Processing**: A critical component that manages long-running and periodic tasks, ensuring eventual consistency with Discogs and processing complex user "intents".
-   **Persistence Layer**: Uses a custom key-value store named `pstore` to persist data as Protobuf messages, enabling structured storage with the flexibility of a NoSQL backend.
-   **Observability**: Fully instrumented with OpenTelemetry for tracing and Prometheus for performance metrics.

## Key Components & Directory Structure

-   `proto/`: Protocol buffer definitions for all services and data models. This is the source of truth for the system's API.
-   `server/`: Implementation of the gRPC and HTTP server logic, organized by functional areas like `wants`, `sales`, and `org`.
-   `db/`: Abstraction layer over `pstore`, providing a clean interface for persisting and retrieving complex objects.
-   `background/`: Contains the logic for background workers that process various types of tasks from the queue.
-   `queuelogic/`: The core logic for the task queue, managing scheduling and execution of background jobs.
-   `admin_cli/`: Command-line interface for administrative overrides and system maintenance.
-   `prober/`: Health checking and monitoring service to ensure system stability.

## Technical Stack

-   **Language**: Go
-   **Communication**: gRPC and HTTP/REST
-   **Data Serialization**: Protocol Buffers (Protobuf)
-   **Database**: custom `pstore` (Key-Value store)
-   **Observability**: OpenTelemetry (Tracing), Prometheus (Metrics)
-   **External Integration**: Discogs API

## Data Flow & Task Processing

1.  **User Interactions**: Users interact with the system via gRPC or HTTP endpoints. These requests often result in immediate actions or the creation of an **Intent**.
2.  **Intent Processing**: When a user wants to change the state of their collection (e.g., move records, update metadata), an intent is stored.
3.  **Queueing**: Background tasks (refreshes, syncs, intent applications) are added to the queue managed by `queuelogic`.
4.  **Worker Execution**: Background workers in the `background/` directory pick up tasks from the queue and perform the necessary work, often interacting with the Discogs API and updating the local database.

## Key Features

-   **Deep Discogs Integration**: Comprehensive syncing of collections, wantlists, and marketplace data.
-   **Stateful History**: Tracks historical changes to records and wantlists, providing insights beyond what the standard Discogs API offers.
-   **Collection Organization**: Support for complex organization rules, snapshots, and moving records between virtual folders.
-   **Sales Management**: Tools for tracking sales, updating prices based on marketplace data, and monitoring sales statistics.
-   **Advanced Wantlists**: Custom logic for managing wants and wantlists with historical tracking and state management.

## Observability & Maintenance

The system is designed for high observability. It exposes a `/metrics` endpoint for Prometheus and uses OpenTelemetry to provide detailed traces of request execution across the server and background workers. The `admin_cli` provides a powerful toolset for manual intervention and state inspection.

## Workflow

You MUST follow the [.agents/workflows/finish.md](.agents/workflows/finish.md) workflow for **EVERY** change, no matter how small. This workflow ensures that changes are committed to a feature branch, pushed, and reviewed correctly. Never push directly to main unless explicitly instructed to do so by the user. Failure to follow this workflow is a breach of project standards.