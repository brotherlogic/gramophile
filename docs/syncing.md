# Syncing Expectations

Gramophile keeps your local collection data matching Discogs by running updates in the background. This page describes what to expect from these background updates.

## Background Updates Queue

Because your collection can be large and Discogs limits how quickly we can request information, Gramophile doesn't update everything instantly. Instead, it places tasks in a background queue to be processed one after the other.

Every task in the queue has:
- **Scheduled Time**: When the task is set to run.
- **Purpose**: A description of what the update is doing (such as updating a record's price or downloading a new addition).
- **Priority**: How quickly the task should be processed relative to others.

## Update Priorities

Gramophile uses three priority levels:
1. **High Priority**: Immediate actions requested by you (such as manual record moves or instant price updates). These run as soon as possible.
2. **Normal Priority**: Routine system operations.
3. **Low Priority**: Long-running or bulk tasks (such as importing a large collection for the first time or checking all wants).

## Syncing Schedule

Once your account is connected and fully imported, Gramophile runs scheduled updates to stay fresh:
- **Full Collection Sync**: Gramophile does a complete refresh of your entire collection from Discogs once a week.
- **Daily Check**: Gramophile checks for any changed records or metadata once a day.

## Handling Discogs Rate Limits

Discogs limits the number of requests external applications can make. Gramophile handles these limits automatically:
- **Rate Limit Cool-downs**: If we hit a Discogs rate limit, Gramophile pauses processing for 1 minute to allow the rate limit to clear before retrying the task.
- **Safety Valve**: If the queue becomes excessively full, low-priority tasks may be skipped to ensure high-priority user actions are not delayed.

## Errors and Automatic Retries

- **Temporary Failures**: If a task fails due to a temporary network issue or server error, Gramophile will wait 5 minutes and automatically try again.
- **Persistent Failures**: If a task continues to fail, the system puts it aside so it can be reviewed and fixed without blocking other updates from running.

