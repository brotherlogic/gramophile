# Organisations Guide

Gramophile helps you plan and organize the physical layout of your vinyl record collection. This guide details how you can configure spaces, group and sort records, and generate instructions for rearranging your physical collection.

## Settings Overview

Your physical organization is configured using a set of rules:

- **Label Ranking**: Rank record labels so that if a release lists multiple labels, Gramophile knows which one to prioritize for sorting.
- **Artist Name Rules**: Clean up or translate artist names for alphabetical sorting (for example, stripping prefixes like "The" or merging spelling variations).
- **Organisations**: Individual setups representing your rooms, shelves, or storage setups.

## Spaces (Physical Shelves & Boxes)

Spaces represent the physical storage where your records are kept:
- **Name**: A friendly name for the space (e.g., "Main Living Room Shelf", "Basement Box 1").
- **Shelves / Units**: The number of rows, shelves, or compartments in that storage unit.
- **Width**: The width of each individual shelf or compartment (defining how much capacity it has).
- **Layout**: How tightly or loosely records should be packed.

## Virtual Folders

You can map specific Discogs folders (such as "For Sale", "Heavy Rotation", or "Archived") to specific physical spaces:
- **Folder**: The name or ID of the Discogs folder.
- **Sorting Rules**: Choose how records are ordered within that folder:
  - **Artist & Year**: Sorted alphabetically by artist name, then chronologically by the release year of the album.
  - **Label & Catalog Number**: Sorted by record label name, then by catalog number.
  - **Release Year**: Sorted by when the album was released.
  - **Date Added**: Sorted by the date you added the record to your collection.

## Record Thickness & Capacity

Gramophile calculates how many records fit on your shelves using different rules:
- **Count**: Every record is assumed to take up the same amount of space (e.g., 1 unit).
- **Disks**: Space is calculated by the number of physical vinyl disks (e.g., a double LP takes up 2 units).
- **Physical Width**: If enabled, Gramophile uses the actual thickness of the record jacket. If a record's thickness is unknown, you can configure Gramophile to ignore the thickness or assume an average thickness based on your other records.

## Grouping and Overflow Rules

- **Grouping**: Group records by artist to ensure that all albums by the same artist stay together on the shelves.
- **Overflow (Spill)**: Define what happens when a shelf or unit runs out of space:
  - **Strict Limit**: Do not allow overflow; if a record doesn't fit, flag the placement as blocked.
  - **Allow Overflow**: Allow records to overflow onto the next shelf, even if it slightly affects the alphabetical order, to maximize storage space.
  - **Look Ahead**: Set how far ahead the overflow logic should check to plan record placements.

## Generating Reorganization Instructions

1. **Take a Snapshot**: Gramophile evaluates your records and current rules to generate a digital layout of where your records should go.
2. **Compare Layouts**: When you update your rules or add new records, Gramophile compares your current layout with the new layout.
3. **Move Instructions**: Gramophile produces a step-by-step list of instructions showing exactly which records to move and where to place them to achieve your target organization.

