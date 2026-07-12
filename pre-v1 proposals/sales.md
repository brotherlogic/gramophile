<!-- Output copied to clipboard! -->

<!-----

Yay, no errors, warnings, or alerts!

Conversion time: 0.548 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β34
* Mon Aug 28 2023 16:56:32 GMT-0700 (PDT)
* Source doc: Gramophile Proposal: Sales Tracking / Adjusting
----->



# Gramophile Proposal: Sales

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Commited</p>



### Abstract

Enables gramophile to handle sales lifecycle


### Process



1. Collection refresh also refreshes existing items For Sale
2. For Sale items are linked to collection entries by release_id
3. Sales are updated at a fixed frequency (measured in days)
4. Sales update follow a strategy
5. Initially one strategy: BOUNDED_LINEAR
    1. BOUNDED_LINEAR tracks from the sale price linearly down to a defined minimum (e.g. Median Price / Lowest price or a multiplier therein)
6. Updates are pushed through the queue
7. Updates are recorded as collection updates


### Field Requirements

N/A


### Execution Location

Sale refresh is handled in the collection sync loop, post sync we do any linking. Linking is random - as in if there are multiple collection entries that link to a sale item, one is chosen at random. Hard refreshing sales (as in removing the sale and re-adding it) may cause it to be linked to another release item in this case. Sale stats are also refreshed each sync looped and connected to the release.

Sale adjustment refresh happens in the sync loop - we look for sales that are in scope for a refresh and adjust sale price in that case. Sale price updates are linear adjustment (as in $$$ of local currency), down to a minimum value which could be (a) fixed (e.g. $5) or (b) set by the sale stats with a multiplier (e.g. twice the lowest value etc.)

Sale data forms part of instance metadata - hence we save changes to that and add an adjustment explainer that turns a sale price change into “Adjusted sale price” type description. Though we do not store the sale price within the metadata block, since it’s unrelated to the actual instance of the record (and not permanent).


### Config requirements

SaleConfig



* SaleStrategy
    * BoundedLinear
        * ReductionFrequency
        * ReductionAmount
        * LowerBound
            * Fixed
            * Value
                * GLOBAL_MEDIAN
                * GLOBAL_MINIMUM
                * GLOBAL_HIGH
            * Multipler (defaults to 1)

Sale IDs are attached to release entries through the release ID. Process is to search through collection to find first instance of given ID and attach - attach is permanent until record is sold

Discogs

Needs to support:

	Sale Creation

	Sale Info

	Order Statuses

Config Validation



* None needed

Moves / Folders Added - only activated if folder movement is active



* “For Sale” Folder
* “Sold” Folder
* Records with attached SaleIDs -> For Sale
* Records with SoldPrice -> Sold


### Milestones:



* Discogs supports Sale Creation
* Discogs supports Sale Info
* Discogs supports Inventory Listing
* Discogs supports Order Statuses
* Config setup and applied
    * Config application creates moves and folders if necessary
* Sale linking in sync loop
    * Confirm sale linkage happens
* Price adjustment in sync loop
    * Validate on a given sale item