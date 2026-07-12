<!-----

Yay, no errors, warnings, or alerts!

Conversion time: 0.437 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β34
* Wed Aug 30 2023 19:53:26 GMT-0700 (PDT)
* Source doc: Gramophile Proposal: Wantlist
----->



# Gramophile Proposal: Wantlist

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Draft</p>



### Abstract

Enables gramophile to handle a list of wants and adds them and removes them as they go


### Process

On a date loop:



1. Add wants as the date block is removed

On a 1x1 loop:



1. Add wants once the previous is bought (PURCHASED)

En Masse:



1. All wants added at once


### Field Requirements

No fields required


### Execution Location

Handled in the want sync loop, we update the wantlist state and then syncs wants with the existing states.


### Config requirements

Wantlist:



* Name string
* StartDate int64
* EndDate int64
* ListType:
    * DATE_BOUNDED - 2
    * ONE_BY_ONE - 1
    * EN_MASSE - 0
* Repeated WantListEntry
    * Id - int64
    * Artist - string
    * Title - string


### Milestones:



* Proposal is added
* Proto supports wantlists
* Gram wantlist new adds a wantlist with params
* Gram wantslist add &lt;name> &lt;id>
* Gram wantlist delete &lt;name> &lt;id>
* Gram wantlist &lt;name> gets the wantlist and want states (with expansion on the id -> id + artist - title