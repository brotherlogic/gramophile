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
- Stylized ASCII Startup Logo: Terminal User Interface features a custom ANSI Shadow ASCII art startup logo styled with Lip Gloss, matching the beerkellar aesthetic.
- Organization Configuration Wizard: A guided interactive Terminal User Interface (TUI) wizard built with Bubble Tea and Charm Huh to help configure physical storage organizations (shelves, boxes) mapping to Discogs folders.
- Organization View: Terminal User Interface mode for viewing physical record organization placement layout and snapshots via `org` / `orgview` command parsing (supporting `--org`, `--slot`, `--hash`, `--debug` flags).
- Resilient Sale Adjustments & Decoupled Refresh: Evaluates sale price reductions resiliently against missing pricing metadata (such as unlinked sales or missing median prices) and decouples sale adjustment processing from catalog refresh.
- AdjustSales Queue Task Handler: Registers a dedicated background queue task handler for `QueueElement_AdjustSales` with deduplication key support to process collection sale price adjustments asynchronously.
- Periodic Sale Adjustment Scheduling: Schedules `AdjustSales` queue tasks periodically in the background validator loop for users with automated price adjustments enabled.
- Sale Adjustment Failure Reporting & Deduplication: Automatically reports unexpected sale adjustment failures to GitHub via `githubridge` with open-issue deduplication to prevent duplicate bug filings for recurring failure conditions.
- Resilient Sales Adjustment Loop: Iterates over user sales resiliently in `AdjustSales`, cleanly handling context cancellation and expected pricing conditions while isolating and reporting individual sale failures without terminating the entire adjustment run.
- Sale Creation Condition Validation & Metadata Persistence: Enforces media and sleeve condition requirements during sale creation in `AddSale`, raising a GitHub issue on missing condition metadata and persisting complete pricing, condition, and timestamp metadata on created sales.
- Sale Condition Synchronization & Backfill: Preserves existing media and sleeve condition metadata on synced sales and backfills missing conditions from Discogs responses in `SyncSales`.
- Multi-Sale Adjustment Resilience Integration Testing: End-to-end integration tests verifying the complete asynchronous sale adjustment lifecycle across multi-sale collections with mixed valid and invalid sale states.
- Record Condition Propagation & Orphan Sale Detection: Automatically propagates media and sleeve condition metadata from local records to linked sales in `HardLink`, and detects active Discogs sales without matching collection records, filing GitHub tracking issues for orphan sales.
- Release Price Statistics Refresh: Updates `GetReleaseStats` integration against modern Discogs structured statistics, accurately recording Low, Median, and High pricing in integer cents and applying fallback defaults on missing or unsold releases.


## TUI (Terminal User Interface)

You can install the Gramophile TUI using standard Go tooling:

```bash
go install github.com/brotherlogic/gramophile/cmd/gramophile@latest
```

To run the TUI once installed, simply execute:

```bash
gramophile
```

## Documentation
- [v1 Requirements](v1/requirements.md): The core feature definitions and user journeys for the v1 release of Gramophile.
