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

## Tasks

1. Define basic update proto
1. Write update remover to remove old updates on change
1. Write update alongside old record change code
1. Add ability to store score updates
1. Add ability to store folder moves
1. Support retrieval of updates in gram get record
