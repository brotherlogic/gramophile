# SLO

Effectively the SLO for gramophile should (a) track
each external entry point, (b) ensure that each such point
performs to some arbritrary criteria and (c) give support
for debugging when we have SLO misses.

## Track each entry point

We have a go interceptor to support this at the API level.
The interceptor does two things:

1. Pre-request: it generates a context key and attaches it to the
   outgoing context. We use the context key to track requests across
   their lifecycle. Context keys are passed through to the queue.
1. Post-request: We have a grafana counter of (a) response code and
   (b) latency for each API request.
1. Also post-request: We log out with the context key a more
   detailed breakdown (structured proto).

We can also implment a helper function to record the process time
within function calls, which are logged as above.

## Criteria

We expect to run at 99.9% on API calls, with all API requests responding within one second
of processing time. The system architecture should support this, since a lot of processing
is async by nature.

## Debugging Support

We should be able to track SLO misses along with the context keys which will assist in
debugging. We need formal storage for this, noting that we don't need to capture successful
requests. We need some kind of longish term log storage, and some form of deeper log
storage and access for requests we want to investigate

Context keys should propogate across both the request, and any queue entries (and sub queue
entries). Validation entries have a seperate entry.

Debugging should track for both API and queue requests - we should also track queue run time
to identify slow queue entries - we can likely relax off the queue time since we do
have Discogs as a bottleneck.

## Requirements

Look at structured log support and access in Loki - there may be a good solution there
that we can build off of. I'd rather keep things Kubernetes native to some extent but ultimately
I'd like a system that I can probe for SLO misses and debug as required. Also look into
tracing support in grafana systems - there may well be a pre-existing solution we can build
off of to get a deeper analysis.

Gemini is recommending grafana Tempo as a tracing solution. Tempo with Open Telemtry looks ideal, may take some work.

## Tasks

1. Setup tempo infrastructure
1. Instrument record adding at the API level
1. Capture the tempo context key and record in logs
1. Build out pieces to support trace viewing
