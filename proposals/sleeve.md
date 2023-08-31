<!-----

Yay, no errors, warnings, or alerts!

Conversion time: 0.49 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β34
* Wed Aug 30 2023 19:48:34 GMT-0700 (PDT)
* Source doc: Gramophile Proposal: Sleeves
----->



# Gramophile Proposal: Sleeves

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Draft</p>



### Abstract

Enables gramophile to track which records have sleeves and what sleeves they have


### Process



1. New config setting to support sleeve tracking
    1. Sleeve types should support multipliers on width
2. Adds a new Field: Sleeve
3. Sleeve types have fixed, specified values and must be one of these
4. Intent push to support setting of sleeve
    2. Must match existing sleeve type
5. Sleeve is updated and saved


### Field Requirements



* Sleeve: string


### Execution Location

Handled as a regular Intent update


### Config requirements

SleeveConfig



* Sleeve
    * Name (string)
    * Mulitplier (float)

Config Validation



* Field “Sleeve” exists


### Milestones:



* Proposal Added
* Add sleeve config setup - setting fails if Field is not present
* Gram sleeve &lt;iid> name - fails if sleeve is invalid
* Intent is processed and sleeve is set
* Mass push from recordcollection