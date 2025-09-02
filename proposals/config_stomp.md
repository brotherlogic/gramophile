# COnfig Stomp

Now that the config is a little more active, and includes some wantlist
changes, we need to make sure that we don't overwrite the config by
accident. This is a proposal to prevent config stomping - i.e. that
config changes are made over the top of the previous, rather than over
the top of an older version.

## Requirements

We don't want to get into a state where we're effectively running a checkout
system, where each checkin is atomic. So (a) we shouldn't expose too much of
the sync system and (b) we should ensure that we're not blocking a change when
we can make it.

### Option 1

Option 1 is that we apply a unique ID to each config version. The ID is required
and if we apply a new config and the gap between the two is bigger than 1, we
reject the application. So the user experience is that we checkout the config,
make some changes, and apply it back. That application fails and we must do the
loop again.

### Option 2

Option 2 is the same as (1) except that we attempt to normalise the diff in someway.
So this process keeps the ID, but the server (a) computes the diff between the proposed
change and the old version. We return that diff to the user, confirm it by asking the user
if the diff looks okay. We then apply the diff to the current version, only failing if there's
a clash somewhere. Thus the CLI provides the glue that ensures that the config chage goes
into place.

### Option 3

We could ignore all of this, and encourage folks to checkout the config, and make a change
over a fresh version.

## Summary

Option 2 is preferred - it is cleaner and requires the fewest user interaction. We can grease
the wheel by running the same loop but not confirming the diff if we don't need to - i.e.
we are applying a new config over the top of the old one.

### Process

For ID we can use an increasing numeral, or the current date in nanos. Perfer the former - it means
we need to keep an atomic save around the config but means we get monotonically increasing values
which makes finding versions a little easier.

## Tasks

1. Add version field into the config
1. Support atomic writes in PStore
1. Support atomic writes in PGStore
1. Have config save increment the version number
1. Server builds out diff if the version is out of whack
1. Gram handles diff exchange with user
1. Server applies diff given chosen change.
