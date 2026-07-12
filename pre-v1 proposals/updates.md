# Updates

## Abstract

This proposal regards the tracking of historical updates to records over time.

## Background

Gramophile acts by reading and making changes to records. Sometimes we would
like to be able to see what changes have been applied to a record over a period
of time - answering questions like, how did it end up in that folder, when did
I listen to this, what score did I give it etc.

We previously recorded updates as a full A->B record change, stored in a proto
update file. The idea there was that we could then read these updates and pull
whatever information we wanted from those updates and answer any questions we
may have from there. Unfortunately this proved problematic since there was
a large number of updates made to a record over time (~2k), and this clogged up
our storage and meant that doing anything useful with these was problematic.

So we decided to remove this method of recording updates and instead streamline the
process a little.

## Proposal

Instead of storing every records change in an update structure we're going to
focus in on key updates. This keeps the update concise enough and gives us control
over what we consider a meaningful update (and gives us the ability to summarize
updates better - e.g. The tracklisting changed, rather than the full diff). However,
this means that as we add new forms of update, we are unable to backfill, instead
we'll build each from scratch. We belive this drawback is far outweighed by the more
positive parts to this proposal.

## Amendment

The problem with this process is that we're adding overhead to cover the cost of
storage and retrieval. Instead we should take a path which mirrors this approach but
gives us recourse to cover updates which we currently don't consider rather than backfilling.

So we should (a) store the proto diff between the old record and the new record on a save.
If the proto diff is empty, we don't save. We then have a process that in the background (or
when needed), pulls the diffs and builds a readable timeline of updates. This should support
dynamic backfill - so if we add new details in the readable timeline, it should automatically
update.

## Tasks

1. Build / Research a proto differ
   1. This will probably need to be custom
1. Apply the differ to record updates
1. Have a seperate process convert the set of diffs into readable changes
1. Ensure these are versioned in code
1. Have a gram get process which pulls updates
1. This should refresh the readable changes if we find a version mismatch
