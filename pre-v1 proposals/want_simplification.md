# Want Simplification

Wants are currently too complex and carry a little too much tech debt because of
our attempt to do hidden wants and handle edge cases in the past. So we're going
to simplify the whole thing by setting some standards:

## Standards

1. All wants *must* be in a wantlist
1. Gramophile supports ad hoc wants in a one-by-one or en masse configuration

## Consequences

Since all wants must be in a wantlist we have two core lists:

1. digital_wants - if we support digital keep this places all digital
   versions of such records into this list. Purchasing any digital version
   removes all digital versions from the list
1. float - this is the list that we use to support ad hoc wants. It can be
   set to EN_MASSE, but is one by one by default
1. When a user adds a want in discogs:
   1. If it is already present in float (in one-by-one), then we signal the want
      status change and let gramophile deal with it. This should not be a surprise
      to end users since they have configured gramophile in this way
   1. digital_wants are en masse by default so you are unable to want an
      existing digital_want
1. When a user deletes a want in discogs:
   1. If it was an existing digital_want, it will get re-added
   1. If it was a float want, we remove it from the wantlist
   1. If it was an entry in one of their configured lists, it'll get picked up
      there and removed (since it's not ready to be wanted yet).

In this way, there is not requirement for wants to be unique to a list, but that
they'll be processed as part of both lists (so it may appear in a float list
before a one-by-one for example).

The float list will appear in the users config. Digital wants do not if they
are computed, but do appear if they are user-specified.

Finally - adjusting the keep status away from DIGITAL means those wants will
be removed from the list.

## Tasks

1. Remove DELETED from the enum for Wants
1. Confirm setting digital_keep adds digital releases to the wantlist
1. Confirm setting digital_keep with release list adds these to the list
1. Confirm that syncing a new want adds it to the float wantlist
1. Confirm that unwanting a want removes it from the float wantlist if present
1. Confirm that syncing a want that's in float with one-by-one, removes it from
   wants
1. Confirm that unwanting a digital_want has no effect
1. Confirm that unwanting an existing listed want as no effect if en-masse
1. Confirm that getting digital wants only returns prescribed wants
1. Confirm that re-upping config with removed digital want removes it from the
   list
1. confirm that manual addition of digital want acts as we expect (i.e adding to
   the list).
1. Confirm that float wantlist is returned and is editable

## Want States

1. PENDING - new wants are in this state
1. RETIRED - something we wanted in the past but no longer want
1. PURCHASED - we bought this one
1. IN_TRANIST - we bought this one, but it hasn't arrived yet
1. WANTED - we want this

## Want syncs

1. Discogs Want Sync
   1. Wantlist state should be adjusted to match reality
   1. Unmatched wants go into float list, maybe even multiple times
1. Wantlist sync
   1. Controls wants on and wants off
1. Culling
   1. Digital wantlist is culled:
      1. Remove things that are no longer digital / digital wants
