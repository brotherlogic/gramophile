# Release Refresh Pruning

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
2023-02-12</p>


<p style="text-align: right">
Draft</p>



### Abstract

Refreshing releases has a synchronization issue - we don’t update the last_refresh_time until the release is refreshed but in the time between adding a refresh release element to the processing queue and the time at which we refresh the date it appears that the release is stale. As a result there is a danger of flooding the queue with refresh requests.


### Options



1. The queue has a cache of refresh times - adding a release refresh updates this cache, running the refresh removes the cache item.
2. Add an extra field to the release scheduled_for_refresh and reject requests where this is greater than the last_refresh_time
3. Periodically prune the queue to remove double adds of release refreshes.
4. Write a marker that indicates a release is to be refreshed on enqueuing, remove the marker once refreshed


### Decisions

We favour option (4) above. 1 is desirable but long term the cache may grow substantially, requiring refactoring somewhere down the line. 2 is appealing but ideally we want to avoid having internal processing markers in the release configuration. 3 doesn’t prevent queue length explosion and could end up being qutie expensive and doesn’t prevent the situation where the queue has grown uncontrollably.

Option 4 is clean, adds a temp file and doesn’t muck up returned protos to the user.


### Proposal

On enqueue where the entry is a refresh release request we write a marker:

```github.com/brotherlogic/gramophile/queue/releaselock/&lt;userid>-&lt;instanceid>```

This is a file with the current nano time contained. If this file already exists, open it and check the timestamp - if the duration since the timestamp is greater than 7 days, replace the file otherwise reject the request. If the existing file cannot be deleted / written, we fail the request with an INTERNAL error.

On dequeue, we delete the marker - this is best effort (i.e. we do not catch any deletion errors).


### Monitoring

The use of these markers should be minimal (i.e. we should not be making excessive refresh requests that cannot be met). To faciliate we add a “intention” field to the refresh_release proto, failing an enqueue if we don’t provide this. We can then monitor failures to enqueue and track both the intention and the time delay. We also monitor add and deletes, in order to catch systemic errors with the marker deletion.


### Milestones



1. This proposal reviewed and added
2. Refresh Release adds intention field and rejects request on missing intention (with monitoring)
3. Markers are added and deleted on refresh release
4. Add monitoring as outlined above
5. Begin to reject requests if they are stale adds