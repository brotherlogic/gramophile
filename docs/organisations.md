# Organisations Guide

Gramophile allows users to define complex physical and virtual rules for organizing their vinyl record collections. This guide details how to configure these rules, referencing definitions in `config.proto` and `organisation.proto`.

## Configuration Structure

Organisations are configured via the `GramophileConfig` protobuf message, under `OrganisationConfig`:

- **Label Ranking**: A list of `LabelWeight` objects used to rank labels in case a release lists multiple record labels.
- **Artist Translation**: A list of `ArtistTranslation` objects that maps artist names to sorted formats (e.g., stripping prefixes like "The" or mapping variations).
- **Organisations**: A list of `Organisation` config records.

## Spaces (Physical Shelves)

Spaces model the physical storage units (such as shelves or boxes) where your records are kept:
- **Name**: A unique identifier for the space.
- **Units**: The number of subdivisions or shelves within the unit.
- **Width**: The physical width of each individual unit (shelf capacity).
- **Layout**: Can be configured as `TIGHT` (compact fit) or `LOOSE` (future support).

## FolderSets (Virtual Folders)

FolderSets map specific Discogs folder IDs into the physical spaces of an organisation:
- **Folder**: The Discogs folder ID.
- **Sort**: The sorting scheme for records in this folder:
  - `ARTIST_YEAR`: Sorted alphabetically by the artist's sort name, then chronologically by release year.
  - `LABEL_CATNO`: Sorted by label name, then catalog number.
  - `RELEASE_YEAR` / `EARLIEST_RELEASE_YEAR`: Sorted by the release date.
  - `ADDITION_DATE`: Sorted by the date the record was added to your collection.

## Density Rules

Density determines how much space a record occupies:
- `COUNT`: Each record consumes exactly 1 unit of space.
- `DISKS`: The space consumed matches the count of physical vinyl disks (e.g., a double LP consumes 2 units).
- `WIDTH`: The physical width of the record jacket. If `WIDTH` is used:
  - The user configuration must have `WidthConfig` enabled.
  - Users can define `missing_width_handling`: `MISSING_WIDTH_IGNORE` (treat missing widths as 0) or `MISSING_WIDTH_AVERAGE` (use the average of other records).

## Grouping and Spill Rules

- **Grouping**: Group records by artist (using `GroupingType` of `GROUPING_GROUP`) to keep albums by the same artist together.
- **Spill**: Dictates behavior when a physical unit runs out of space:
  - `SPILL_NO_SPILL`: The placement fails, blocking movements.
  - `SPILL_BREAK_ORDERING`: Allows records to overflow into the next shelf unit, potentially breaking alphabetical sorting to maximize storage density.
  - **Look Ahead**: Specifies how far ahead the spill logic looks (`-1` for infinite).

## Snapshots & Auditing Moves

1. **Snapshot Generation**: The system evaluates your record collection and generates an `OrganisationSnapshot`.
2. **De-duplication & Hashing**: Each snapshot contains placements (`iid`, `unit`, `index`, `space`, `width`, `sort_key`) and is hashed using SHA-1.
3. **Move Diffs**: By comparing two snapshots, Gramophile generates a `SnapshotDiff` detailing the moves required to reorganize your physical collection.
