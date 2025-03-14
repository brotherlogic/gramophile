<!-----

Conversion time: 0.367 seconds.

Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β39
* Tue Oct 01 2024 06:21:07 GMT-0700 (PDT)
* Source doc: Stats Tracker for Gramophile
----->

# Stats Tracker for Gramophile

<p style="text-align: right">
Brotherlogic</p>

<p style="text-align: right">
Draft</p>

<p style="text-align: right">
</p>

### Abstract

A system for supporting stats on the record collection irrespective of requests etc.

### Background

Gramophile monitoring is highly server based, i.e. we can track on requests, but we can’t track overall statistics about the system through the server job, nor should we. We propose here to add a stats endpoint for the API and use this to build graphing.

We build out the stats endpoint which collects basic statistics about the collection as a whole and then add a job which calls this endpoing every minute or so and exports the data to prometheus/grafana. This way we can support per user dashboarding without having to provide those dashboards ourselves - folks can jsut run this infrastructure on their own accord when they want to have collection dashboarding.

We may choose to shard this data over time but for now we’ll start with some very basic stats:

1. Collection Size
    1. Active
    2. Inactive (i.e. sold)
2. Total sales
    3. Year

We run this as a non-gramophile job in the cluster that exports this data to prometheus.

### Steps

1. Add this doc to the repo
2. Configure the basic info in the proto, along with the API calls
3. Support the API
4. Add exporter job to cluster
5. Link the exporter job to grafana and build out dashboard
6. Store dashboard with exporter job
