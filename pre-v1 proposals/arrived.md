<!-----

Yay, no errors, warnings, or alerts!

Conversion time: 0.339 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β34
* Tue Aug 29 2023 21:09:47 GMT-0700 (PDT)
* Source doc: Gramophile Proposal: Arrived Date
----->



# Gramophile Proposal: Arrived Date

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Published</p>



### Abstract

Enables gramophile to track when records arrive, as well as addition date


### Process

The idea here is that we add a record through the Discogs UI, or through gram add when we purchase the record and then we set the record as arrived once it arrives. The reason we do this is to support want list processing principally. For example, we may buy something off our wantlist, but it’s not arrived yet so not ready for cleaning / width / filling or listening to. But we do want that record to be deleted from the wantlist in the interim. Hence we have an extra field called Arrived that captures when the record is in the hands of the buyer and we can use that field for movement. In RC I hold a record in “Limbo” once it’s added and move it out of Limbo once it’s arrived and then it goes off for cleaning.


### Field Requirements



* Arrived: string (date string)


### Execution Location

Handled as a regular Intent update


### Config requirements

ArrivedConfig



* Mandated

Config Validation



* Field “Arrived” exists


### Move Additions

If we have auto move enabled:



* Additions without arrived date are placed into Limbo folder
* Once arrived, records are placed into “Listening Pile” folder


### Milestones:



* Proposal Added
* Add arrived config setup - setting fails if Field is not present
* Gram arrived &lt;iid>
* Intent is processed and arrived is set
* Mass push from recordcollection