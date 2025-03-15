# Cancel Wants

Sometimes the backlog is too great, and we need to make sure that we're not
over buying, feeding the backlog and putting ourselves in a bind in the future.
So the idea here is that we mark some folders as explicit checks agsint that, define
a count threshold on those and then disable wantlists if those counts are broken.

## Process

In org config we can mark folders as "Listening", and then in wantlist config we
can define a listening threshold - a number of which the total number of items
in the listening folders must be below in order for a wantlist to be active.
