<!-- Copy and paste the converted output. -->

<!-----

Yay, no errors, warnings, or alerts!

Conversion time: 0.426 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β34
* Wed Aug 30 2023 19:52:51 GMT-0700 (PDT)
* Source doc: Gramophile Proposal: Wants
----->



# Gramophile Proposal: Wants

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Draft</p>



### Abstract

Enables gramophile to handle wants.


### Process



1. Wants are pulled in the sync loop
2. Gram want add id adds a want
3. Gram want delete id deletes a want
4. Gram want lists the current wants


### Field Requirements

No fields required


### Execution Location

Handled in the same loop as record refresh. Similar process, we pull pages of wants and save them out. Same delete mechanism of syncing with existing wants, with the caveat that RETIRED and PURCHASED wants are not deleted, but maintained


### Config requirements

Want:



* Id - int64
* want_added_date - int64
* WantState:
    * WANTED
    * RETIRED
    * PURCHASED


### Milestones:



* Config added to support wants
* Discogs supports want listing
* Discogs supports want adding
* Discogs supports want deletion
* Gram want add adds a want
* Gram want delete deletes a want
* Gram want lists the existing wants - maybe paginate if we need to