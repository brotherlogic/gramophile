# gramophile

Gramophile is a system for managing your record collection through Discogs.
Gramophile is not a system to support business selling. It is still under development.

## Prober

Probers validate the login process and rebuild the user db.

## Development

Gramophile is built with Go.
Current Go version: 1.26.2

## Features
- LocateRecord: Provides functionality to query the location of records within the collection via the `gram locate` CLI command. It displays the artist name along with the title, and the shelf width percentage where the record is located.
- Organization Configuration Wizard: A guided interactive Terminal User Interface (TUI) wizard built with Bubble Tea and Charm Huh to help configure physical storage organizations (shelves, boxes) mapping to Discogs folders.
- Organization View: Terminal User Interface mode for viewing physical record organization placement layout and snapshots via `org` / `orgview` command parsing (supporting `--org`, `--slot`, `--hash`, `--debug` flags).
- Resilient Sale Adjustments & Decoupled Refresh: Evaluates sale price reductions resiliently against missing pricing metadata (such as unlinked sales or missing median prices) and decouples sale adjustment processing from catalog refresh.
- AdjustSales Queue Task Handler: Registers a dedicated background queue task handler for `QueueElement_AdjustSales` with deduplication key support to process collection sale price adjustments asynchronously.
- Periodic Sale Adjustment Scheduling: Schedules `AdjustSales` queue tasks periodically in the background validator loop for users with automated price adjustments enabled.
- Sale Adjustment Failure Reporting & Deduplication: Automatically reports unexpected sale adjustment failures to GitHub via `githubridge` with open-issue deduplication to prevent duplicate bug filings for recurring failure conditions.


## TUI (Terminal User Interface)

You can install the Gramophile TUI using standard Go tooling:

```bash
go install github.com/brotherlogic/gramophile/cmd/tui@latest
```

To run the TUI once installed, simply execute:

```bash
tui
```

## Documentation
- [v1 Requirements](v1/requirements.md): The core feature definitions and user journeys for the v1 release of Gramophile.
