<!-----

Conversion time: 0.282 seconds.

Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β37
* Fri Aug 09 2024 16:23:56 GMT-0700 (PDT)
* Source doc: Queue Priority
----->

# Queue Priority

<p style="text-align: right">
Brotherlogic</p>

<p style="text-align: right">
</p>

<p style="text-align: right">
Draft</p>

### Abstract

Defines how to handle priority in the processing queue, to support better managing of incoming requests

### Background

The gramophile work queue mediates between the need for a user response and the fact that Discogs can pushback on a change we have made. In addition there are legitimate background tasks that we would like to do periodically (e.g. update a release, refresh a property of a given release etc.). We currently manage the different tasks by adjusting the run date of urgent tasks to ensure that they run before any backlog. This document proposes using a priority system to support better work management in the queue.

### Priority

We propose to add a prority to each queue entry - PRIORITY_HIGH and PRIORITY_LOW. User driven requests will be given HIGH, and background jobs will be given LOW. When picking an entry from the queue we will scan for HIGHS and then pick off the queue if we can’t find one. In addition to tracking queue length we will also track queue idle time - i.e. the time at which each item was sat before being processed. Split out by priority should show that the system is working as expected and also if there is a regression in queue processing time.

### Process

1. Add priority enum to queue proto and to queue entry (PRIORITY_UNKNOWN, PRIORITY_LOW and PRIORITY_HIGH)
2. Record queue addition time on Enqueue
3. Add queue processing time, split out by priority value to dashboard
4. Getnextentry should scan for a high priority item first, then pick the next item from the queue
5. Remove time offset of enqueue, instead pass through the right priority

<!-- watermark --><div style="background-color:#FFFFFF"><p style="color:#FFFFFF; font-size: 1px">gd2md-html: xyzzy Fri Aug 09 2024</p></div>
