# Domain Context: Gramophile

Gramophile is a stateful integration layer for Discogs, designed to enhance the experience of managing a vinyl record collection through background task orchestration and historical tracking.

## Ubiquitous Language

- **Record**: A specific instance of a vinyl release in a user's collection, tracked via a Discogs Instance ID (IID).
- **Want**: A desired release, tracked via a Discogs Release ID or Master ID.
- **Wantlist**: A curated collection of wants, which can be custom-defined within Gramophile.
- **Intent**: A user's desired future state for a record or want (e.g., "move to folder X", "set price to Y"), processed asynchronously.
- **Queue**: The background task runner that processes intentions, collection refreshes, and marketplace updates.
- **Deep Dispatcher**: The architectural pattern where the queue acts as a pure orchestration layer, delegating business logic to a task registry.
- **Seam**: A formal boundary or interface (like `TaskHandler`) that allows for generic handling of specific implementation details like deduplication or validation.

## System Boundaries

- **Discogs API**: The external source of truth for collection state and marketplace data.
- **PStore**: The custom key-value persistence layer where Gramophile stores its stateful history and proto-based models.
- **Background Runner**: The service responsible for executing tasks from the queue and interacting with external APIs.
