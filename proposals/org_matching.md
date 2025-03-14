<!-----



Conversion time: 0.308 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β36
* Mon Jun 17 2024 06:32:57 GMT-0700 (PDT)
* Source doc: Org mataching
----->



# Org Matching

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
2023-02-29</p>


<p style="text-align: right">
Draft</p>


Abstract

This proposal outlines a process for matching the existing org system (from recordsograniser) with what Gramophile is able to do.


### Background

Recordsorganiser is the existing piece of infrastructure I use to organise records. When a record is moved to a folder which is in scope (pretty much any folder), it pulls all the records from a given org and re-orders them. Recordmover then picks up the move and the new record location and prints out the move to the printer. Recordsorganiser is similar to gramophile org in many ways - however it differs in other ways and I would like the two to be somewhat aligned before considering gramophile org feature complete.


### Process

The process here has two factors: (1) To align the two organization systems and ensure they’re in sync but also (2) not back align - i.e. not fix issues with recordsorganiser, but instead just ignore them

Write a script that (a) pulls the latest org from each source and builds a list of index -> instance_id for each. It should then go through and alert on significant discrepencies between the two (i.e. places where the index is different between the two by some threshold). Rorg has a different model of capturing catalogue numbers then gramophile, so this is a place where we may see some local disagreement. We should address this local disagreement where we need to but also be able to ignore it


### Milestones



1. Add this proposal to repo
2. For two given locations
    1. Assume that the system is org’d from rorg.
    2. Script pulls organisation from Rorg
    3. Script pulls organisation from Gramophile
    4. Indexes are built
    5. Comparison is made - prints the indices of the diff
    6. Script supports an allow listed file - i.e. records we know gram has correctly sorted and is ignored from the comparison. (index numbers are shuffled here).
    7. Have the script run and raise issues on the basis of a diff
    8. Crontab re-run and adjustment