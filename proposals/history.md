# History

We would like gramophile to maintain a history of changes. Both changes resulting
from downstream discogs edits, and of changes made by us or by gramophile itself.

## Background

Our first attempt at tracking history was to store the old and new record each
time we detected a change, and then use that to derive a history of changes. This
was promising assuming that (a) we had a lot of storage and (b) that our ability
to suck in those changes was fast. Only (a) was true, and (b) caused us to slow
down dramatically.

The advantage of this approach was that we could retrospectively build out
a change set given new criteria. But there are actually only a few things that we
genuinely care about when doing that, so this advantage is diminished somewhat.

## Proposal

We use the same logic entry - i.e. when saving a record we compare it with the
version we have on disk and look for changes. When we find a change we add it
to a changelog file that's stored next to the record itself. If we want to
capture new changes we need to add it to that list.

We can avoid the cold start problem outlined above by additionally storing an one
year archive of each save in a temp directory. We have a controlling file which outlines
the last time each change was captured for a given record. If we find that we're
doing a change set for the first time, we add a background job which will process the
saved files, build out the changes and update the control file.

The changelog file will be a bundled set of N changes, once the changelog file is full
we create a new one, this bookkeeping is tracked in the control file.

We'll seed this with tracking width over time.

## Tasks

1. ~~Add a path to also save a historical version each time we save a record~~
1. ~~Add control file proto to the proto list~~
1. ~~Add changelog file proto to the proto list~~
1. ~~Add logic to save new widths to the changelog if we see a difference between
   the old file and the new, updating the control file if it's not zero
1. ~~Add background job to process historical versions and build out the changelog,
   and update the control file. Background job should not run if control file is set~~
1. Kick off the background job when we encounter a zero last in the control file
1. Add get history piece to gram get, showing changes over time
