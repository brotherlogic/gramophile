# Add

Define an API to add / remove records from gram

## Background

In recordcollection we supported addition through a seperate service called recordadder.
The purpose here was to slowly feed in records into the collection to prevent swamping, i.e.
that we end up with lots of things to listen to all at once, and miss the time to listen
to the existing collection. Here, this is uncessary since we already have good metrics around
when we last listened to something etc.  So the goal here is to allow us to add and delete
records from the collection via the CLI.

## Addition

For addition we need to specifiy:

1. The discogs ID
1. The purchase price
1. The purchase location
1. The goal folder

Where the purchase price must be fully specifed (i.e. dollars and cents in the US). The purchase
location should be largely freeform but we should support altering this post hoc to allow
for corrections to e.g. spelling mistakes. The goal folder should be specified if we have
that piece enabled in the config.

This then adds the record and places it in a default folder (user specified). This folder should
exist in the collection (we should validate this on each addition).

## Deletion

Deleting is done through instance id only. It removes refs from the collection and calls the delete
endpoint to remove the record.

## Tasks

1. Setup addition config - spec default folder and enable the endpoint
1. Add addition API endpoint, checks for the default folder, adds the record
1. Add delete API endpoint, removes from Discogs, removes from collection
