<!-----



Conversion time: 0.412 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β35
* Mon Mar 18 2024 06:40:54 GMT-0700 (PDT)
* Source doc: Digital Wants
----->



# Digital Wants

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
2024-03-18</p>


<p style="text-align: right">
Draft</p>



### Abstract

Describes a process for digital wanting an existing record, and removing said wants once one of them is purchased.


### Background

Sometimes we want a particular type of release, but have no real skin as to which particular release we’d like. For example, we may want a pre-1980 vinyl version of something, the most comprehensive version of a release (i.e. the one with the most tracks), or we may just want a digital copy of any type. This proposal is about making this inference a first class object, and therby allowing us to support digital wants.

Gramohile supports the option to set a digital keep option for a given release we own. The idea here is communicating to gramophile that we like this record, but would be happy with a digital version - kind of like minitng up but with CD/Digital releases.


### Proposal

We will transition wantlists to additionally support master releases with a given filter. Wantlists can either set a filter on the whole thing, or on individual entries - individual entries overridng the global settings. Then in the wants check loop if any of the entries have been purchased the whole set is considered purchased. Master wants are periodically refreshed (every week) to evaluate if new entries have been added to the master and are in scope for the want.

Master want filtering is the standard gramophile release filtering construct. If the system tries to add a purely digital release to the list, an issue is raised that this release is available digitally, and the system carries on regardless

Digital want construction is one-by-one built list from any release that has a keep status of KEEP_DIGITAL. Master releases for the specified release are added to the wantlist in the collection refresh loop.


### Milestones



1. This proposal is added to the repository
2. Wantlist proto update to add master id option and filter spec (global and local)
3. Want supports master setting - fans out to child wants (through queue)
4. Master want is cleared if subwant is purchased
5. Proto config to support digital wantlist handling
6. Recordcollection loop adds elements to digital want list (creating if necessary)
7. Profit