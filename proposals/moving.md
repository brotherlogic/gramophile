# Moving Records

Describes how we move records around as part of the organisation
loop of gramophile. Effectively we use the classification as a hint of
the lifecycle, and properties of the record to determine where it should
be placed.

## Background

Our initial thoughts around movement was that we should use conditional
rules in order to move records - effectively combining the classification
and movement steps we used in record collection. At the time we wrote
[folder_movements](folder_movements.md) this was my thinking - that
the two sided approach was hard to work with and meant that we were
thinking about record movement in two places. In retrospect I believe
that this is actually the correct model to use - that we say something
about a record and then that thing (and other properties) define where
it should go. Though the two aren't seperate, I think it gets complex to
talk about them combined. Like "This record needs to be cleaed" and "It's
a 45, so I place it here so it's ready to be cleaned" are two disjoint
thoughts.

That being said both steps require some filtering. For example we may have
a rule which says ```is_uncleaned -> to_be_cleaned``` and then a move rule
which says ```to_be_cleaned: is_seven_inch -> seven_inch_cleaning_pile```.
So both sides have filtering rules, it's just that we keep it simple by
defining each side of the rule in a seperate place. This goes some way
to prevent complexity since if we have multiple rules defining classfication
and mulitple rules defining movement then we have ```x*y``` rules defining
the move.

## Movement

We move based on format. We can keep formats limited, but we would like to
name them. We have a basic set:

```
contains 12 || contains LP -> 12 Inch
contains 7 -> 7 Inch
contains CD -> CD
contains File -> FIle
```

Format logic is applied to top to bottom with first matching rule being used.
This makes sense somewhat - a 12" record that also contains a 7 Inch is typically
filled with the 12s, and a 7inch that contains a CD is typically filled with the
7 inches.

Boxsets place a wrinkle in this since they tend to be of variable sizes and there's
no immediate way to automatically place them. We work around this by suggesting
that oversized records (i.e. box sets that do not fit on their respective shelves, or
that we do not want to fit there), are flagged as such and handled seperately. You can
file them as you would 12 inches, or just ignore the filling and put them wherever
fits for your collection.

For simplicity size is set on a record and assumed to be regular from the outset. Thus
size is expressed as an enum:

```
SIZE_REGULAR
SIZE_OVERSISED
```

Thereby allowing us to handle these. A gram update step allows us to mark records as either
regular or oversized, or leave as is.

## Movement Rules

Internally we have preset classification rules as outlined above, but we support overriding
these with custom rules that are applied prior to the internal ones. These are specified in the
same manner as classification rules.

The movement rules say where a record should be moved given the classification, e.g.:

```
to_be_cleaned && 12_Inch -> 12 Inch Cleaning Pile
```

A couple of validation rules apply here. One is that a classification *must* exist for the move
rule to exist. Deleting the classification, or renaming the classficiation here will cause
validation to fail. Furthermore the destination *must* exist.

Movement is made in the post update loop - i.e. anytime a records classification changes, new
rules are applied. We move solely on the basic of classification changes.

## Tasks

1. Write format logic proto
1. Write logic to postpend internal movement rules on config update
1. Write logic proto to capture overall movement
1. Write validation rules for changes to config
1. Refactor move loops to consider classification changes only
1. Refactor move loops to apply our given rules and move records accordingly.
