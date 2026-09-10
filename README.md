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
- Stylized ASCII Logo Header: Terminal User Interface features a custom ANSI Shadow ASCII art logo styled with Lip Gloss that persists across all views throughout the application session.
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
- Raised Max User Queue Size: Expanded per-user throttlable background task queue limit to 200 items.
- Release Price Statistics Refresh: Updates `GetReleaseStats` integration against modern Discogs structured statistics, accurately recording Low, Median, and High pricing in integer cents and applying fallback defaults on missing or unsold releases.
- Admin Approval Username Display: Displays the authenticated user's Discogs username on the waitlist status screen while waiting for administrator approval in the Gramophile TUI.
- Sleeve Condition Propagation & End-to-End Sale Condition Preservation: Propagates sleeve condition metadata in `AdjustSales` when enqueuing price update queue tasks, and verifies complete end-to-end condition preservation through the `AddSale` -> `HardLink` -> `AdjustSales` pipeline with comprehensive integration tests.
- Post-Median Reduction Cycle Calculation: Accurately calculates post-median price reduction cycles using immediate first-cycle triggering once post-median time has elapsed and adds zero-frequency guards to prevent division by zero in sale adjustments.
- Post-Median Sale Reduction Integration Testing: Comprehensive end-to-end integration tests verifying the full asynchronous post-median price adjustment workflow, including immediate first-cycle reduction, multi-cycle intervals, static and Discogs Low lower bound constraints, holding states, and queue payload verification.
- Incremental Listing Sync & Early Termination: Refactors `SyncSales` to support incremental sales sync with early termination when encountering previously synced listings sorted in reverse chronological order.
- Incremental Order Sync Schema: Defines `SyncOrders` queue task element and adds `last_order_sync` timestamp tracking to `StoredUser` for incremental Discogs order syncing.
- Periodic Order Sync Scheduling & Admin CLI: Schedules `SyncOrders` queue tasks periodically in the user validator loop for active users and provides `syncorders` admin CLI command to manually enqueue order synchronization tasks.
- Waitlist User State Display & Live Bypass: Displays the authenticated user's `UserState` enum value during the waitlist admin approval screen in the Gramophile TUI, and bypasses the waitlist screen directly to the main application for live users.
- Main Application Command Input Bar: Terminal User Interface provides an interactive command input bar in the main application state to issue commands (`locate <release_id>`, `org [name]`, `configure`, `quit`) with inline error reporting and command execution.
- Flagless & Multi-Word Organization Search: The TUI allows viewing organizations directly via `org [name]` without requiring the `--org` flag, correctly preserving multi-word organization names (e.g., `org 12 Inches`) and defaulting to the configured organization when run without arguments.
- Interactive Configuration Command & Selection: Supports the core `configure` command in the TUI main application which opens an interactive selection menu with `org` configuration, as well as the `configure org` direct shortcut.
- Streamlined Main Application Screen: Cleared the transitional "Handoff to main application complete" placeholder from the TUI main application view for a cleaner command interface.
- Toggleable Command Help in TUI: The main application view hides the command list by default and allows users to toggle the list of commands on and off by pressing "h", replacing the static footer with contextual "press h for help" / "press h to hide help" guidance.
- Streamlined Organization View Layout: Formats organization placements cleanly as `[<index>] <artist - title> [ <shelf> / <slot> ]` (e.g. `[1] Andrew Bird - Are You Serious [ Main Shelves / 1]`), omitting verbose labels and width metrics, and streamlines the view to display the organization hash in the header without redundant organization header lines.
- GetRecord All Records Schema: Extends the `GetRecordRequest` protobuf message with `bool get_all_records = 7` in the request oneof to support retrieving all records in a collection.
- Animated Spinner on Organization View Loading: Replaces static "Loading..." placeholders with an animated Bubble Tea spinner in the TUI organization view while record details are asynchronously loaded for placements.



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
