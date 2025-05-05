# Sell

Gramophile should support autosell via the "sell" directive.

## Background

In recordcollection sales are handled automatically on a spill in a given collection.
For gramophile we'd like to support this too - the first step to enabling this is to
define a sell directive.

The sell should list the record for sale, setting all the fields from internal settings, or using
provided overrides. Default sale price should be set from either the recorded max sale with
an optional buffer, or from the recommended price for a given condition, again with an
optional buffer.

To leech into recordcollection, when we do a sync, we should also pull in the current sale id.
We don't want to record this as a specific link, so trigger a gram update and have grambridge
pull in the linked sale id, and perform a rc update if necessary.

## Config

```
message SaeConfig {
    Enabled enabled = 1;

    float32 listing_price_buffer = 2;
    enum ListingStrategy {
        LISTING_STRATEGY_SPECIFY = 0;
        LISTING_STRATEGY_HIGH = 1;
        LISTING_STRATEGY_MEDIAN = 2;
        LISTING_STRATEGY_RECOMMENDED_MINT = 3;
        LISTING_STRATEGY_RECOMMENDED_VGPLUS = 4;
    }
    ListingStrategy listing_strategy = 3;

    bool allow_offers = 4;
}
```

Listing fields:

1. release_id -> set from instance
1. condition -> set from insstance
1. sleeve_condition -> set from instance
1. price -> passed in / set from strat
1. comments -> passed in
1. allow_offers -> passed in / from config
1. status -> set ("For Sale") / passed in (--draft)
1. external_id -> passed in
1. location -> passed in
1. weight -> set from instance / passed in / "auto"
1. format_quantity -> passed in / "auto"

## Tasks

1. Set sale config up
1. Discogs|Enable record sales
1. Gramophile supports parameterized sales - saving the sale with a direct link to the listed sale
1. Gramophile sets price from settings if available
1. Gramophile refresh record returns linked sale id if available
1. grambridge|Grambridge sets the saleid if returned
