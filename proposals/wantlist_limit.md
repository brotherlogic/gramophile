<!-----



Conversion time: 0.359 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β36
* Mon Jun 17 2024 06:33:52 GMT-0700 (PDT)
* Source doc: Limited Wantlists
----->



# Limited Wantlists

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Draft</p>



### Abstract

This doc describes the process of limiting the amount of tracked wantlists, so that the size of the wantlist stays small


### Background

Gramophile supports wantlists - a list of records that the processor works through them in a given way. We would like to support a large number of lists, but have the system pick and choose between them as we go through them. The problem here is you end up with a large number of lists and it’s diddicult to work through your wantlist in that scenario.


### Process

We add a field to each wantlist that tracks the last purchase date (derived from the additin date from the record). We also specify a field on the wantlsit config that limits the number of active wantlists. We then track wantlist activeness and keep it limited to the number, using the last purchase date as the ordering. As items are ticked off the list, it adjusts the purchase date and so the lists will naturally cycle. We also add a “arrived” field for a record to distringuish between a record that’s purchased and one that is in the collection. This gives us a natural waiting period between wantlist cycles. This doesn’t necessarily limit the number of wants since we do allow master expansion.


### Milestones



1. Add this doc to the repo
2. Add wantlist field to track last purchase date
3. Add wantlist config setting to support max_active_lists (defaults to 3)
4. Add arrived field to a record
5. Support setting the arrived field
6. Translation layer between gramophile arrived and recordcollection arrived
7. Wantlist has a in between WANTED -> PURCHASED state
8. Wantlist supports removing item on WANTED -> NEW_STATE
9. Wantlist supports adjusting purchase date once PURCHASED
10. Active setting added to wantlist
11. Unactive wantlists are not processed and wants are removed
12. Wantlist adjustment adjusts active setting on update