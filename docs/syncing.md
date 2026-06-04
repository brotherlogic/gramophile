# Syncing Expectations

Gramophile maintains eventual consistency with Discogs by processing data-heavy and external API tasks asynchronously via a persistent background queue.

## Asynchronous Queue Processing

Tasks are stored in the database (`pstore`) as `QueueElement` messages under the `gramophile/taskqueue/` prefix. A background worker loop (`queuelogic.Run`) processes these tasks sequentially.

Key properties of each task include:
- **Run Date**: A timestamp representing when the task is scheduled to run. If this is in the future, the worker will sleep until it is reached.
- **Intention**: A human-readable description of why the task was queued.
- **Priority**: Detrmines the order in which items are fetched.

## Queue Priority Levels

Gramophile uses three active priority levels:
1. `PRIORITY_HIGH`: Reserved for critical, user-triggered changes that should execute immediately (e.g., immediate price updates, manual movement of records).
2. `PRIORITY_NORMAL`: Default priority for system operations.
3. `PRIORITY_LOW`: Used for long-running, bulk operations (e.g., initial user collection refresh, large wants updates).

## Background Sync Intervals

Once your account is live, Gramophile executes routine background refreshes:
- **Full Collection Refresh**: Refreshes your entire collection from Discogs once a week (`time.Hour * 24 * 7`).
- **Collection Check**: Checks through the database for metadata and sync tasks once a day (`time.Hour * 24`).

## Throttling and Rate Limiting

Discogs enforces rate limits on external API requests. Gramophile handles rate limits gracefully:
- **Throttling Detection**: If a request fails with a `codes.ResourceExhausted` error, the system checks the error reason.
- **Queue Limits**: If the limit is due to the queue being full or user queue limits being exceeded (`User queue limit reached` or `Queue is full`), the task is deleted/dropped to protect resources.
- **Cooldown**: For standard API rate limits, the worker sleeps for 1 minute to allow tokens to regenerate, leaving the task in the queue for a retry.

## Error Handling and DLQ

- **Internal Errors**: If a task fails with a `codes.Internal` error, the system automatically files a GitHub issue on the `brotherlogic/gramophile` repository, deletes the current entry, and re-queues it with a 5-minute backoff.
- **Dead Letter Queue (DLQ)**: For all other non-recoverable errors, the item is moved to the DLQ under the database prefix `gramophile/dlq/` for manual administrative intervention.
