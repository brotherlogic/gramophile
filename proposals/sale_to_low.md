<!-----



Conversion time: 0.515 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β35
* Tue Jan 09 2024 06:24:57 GMT-0800 (PST)
* Source doc: Sales To Lower Bound
----->



# Sales to Lower Bound

<p style="text-align: right">
brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Published</p>



### Abstract

An extension to median set pricing which will reduce the price to a lower bound over a period of time. The idea here is that for the REDUCE_TO_MEDIAN strategy we reduce the sale price from the high to the median value, hold it there for a period of time, and then reduce it further - ideally getting a sale.


### Goals



1. Once a sale has been at the median price for a period of time, we start reducing it to a minimum.


### Configuration



1. Applies to “REDUCE_TO_MEDIAN” sale strategy without adjustment
2. Extra Config:
    1. Post_median_time - the time to wait before reducing
    2. Post_median_reduction - the amount to reduce by
    3. Post_median_reduction_frequency - the time cycle to perform a reduction
    4. lower_bound_strategy : {STATIC,DISCOGS_LOW} enum that sets the lower bound
    5. Lower_bound - int32; if lower_bound, ignored if DISCOGS_LOW is set
3. Record Metadata
    6. Time_at_median = the time a sale has reached the median price


### Application



1. If the post_median_time is greater than zero then the reduction strategy applies
2. Additional configuration of time_at_median (doesn’t reset when the median changes)
3. Computes price as:
    1. Post_median_cycles = since(post_median_time) / post_median_reduction_frequency
    2. New_price = max(median-post_median_cycles*post_median_reduction, lower_bound)
4. Lower_bound can be set explicitly (e.g. $5), or can be set to be discogs reported LOW value


### Milestones



1. Low value is extracted and stored on refresh cycle
2. Config is adjusted to support new values
3. Time_at_median is set on record metadata once the sale price is at the median (reported at the time)
4. New computation is set from the above


### Rollout



1. Initially applies to a single sale item (force set)
2. Validate that reduction works as expected
3. Applies across all sales (check that not setting config values works)
4. Applies across current sales