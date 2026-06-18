# Sale Matching

When we look through our sale inventory, we only match when the inventory item is "For
Sale" or "Sold". And we don't match sale ids that are already linked in the database.

Matching sale ids to instance ids is tricky in the edge cases. Typically we
have one version of a release and then matching is easy - we just link directly. If we
have multiple inventory items with the same release id and they are each "For Sale", that
indicates a bug, and we should file an issue to fix. If we have mulitple inventory items
but only one is "For Sale" we just link that entry and ignore the ones that are
marked "Sold"

The reason we do this, is that we can often sell an item - i.e. Discogs marks the release
as sold, but then the sale falls through, or the record is returned is relisted.

## Multiple instance ids

We have a few cases where we have multiple instance ids tied to a single release id. If
we are in this scenario we have to ask the user to resolve the match - we file an issue
with the title "Sale Linking" and the issue body lists the instance ids and the sale ids (or ids).

## Sale state changes

If we find a release with a linked sale and that sale is marked as a category which is not "For Sale" or "Sold", or the linked sale has been deleted, we remove the linking. This resets everything, even the fixed ids.

Note that each sale id can only be linked to one instance id - the manual linking should fail if we try to link a sale id to a given instance id that already has a linked sale id.