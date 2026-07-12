# Score History

## Background

In recordcollection we have the ability to track the record scores over time.
We should have the same for gramophile, but we can do this better. In recordcollection
a listen and a score are naturally tied together but in practice we sometimes want to
score a record with a partial listen (i.e. one side of a 12"), or without a listen at
all. Usually this happens due to a mismatch between sale expectations - so a record
was listed to sell which we knew was a keeper.

So in gramophile we should both (a) record scores over time in the record metadata but also
(b) support a non-listen score. We should also be able to backtrace scores from recordcollection
into the grampophile DB.

Finally we have tended to struggle a little with the 5 point scoring system that Discogs
supports - it's a little too narrow to support a full range of scores and we tend to use it 
poorly with 1,2,3  being potential sales and 4,5 being potential keepers. In gramophile it would
be nice to support larger score ranges so we can also add this whilst mucking with scoring.

## Approach

We first amend the score to be a proto with fields that reflect (a) the given score, (b) the
translated discogs score (i.e. a mapping from the given score down to 1-5), (c) the date of
scoring and (d) the type of scoring: UNKNONW, NON-LISTEN, LISTEN.

We then adjust gram score to be support the new range and an additional flag to indicate that
this was a non-listen score. On a non-listen score we don't update the last listened timestamp.

To transition between the old score and the new score, we'll deprecate the old scoring field and
move directly to the new one.

Finally we'll transition the record colleciton scores en masse, all set to non-listen scores in
order to support the coming year of relistening.

## Tasks

1. Rewrite metadata score structure
1. Add score config into overall config structure
1. Gram score supports full range scoring
1. Gram scores are recorded in record metadata
1. gram score supports backdated scores