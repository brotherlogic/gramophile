# Wantlists Storage

In Gramophile we store wantlists in the database. This is fine for our needs, but does mean
that in data loss scenarios we lose our wants. And the principle of gramophile is that
we store state in fields, config in the config file and history in the database.

Thus this proposal is to move wantlist storage out of the database and into config.
This does raise some problems since we need to be careful about what is stored in the list,
that we ref only core information, and derive the remainder and do a sensible merge when we
blend the two together.

## Proposal

Our proposal is to add some elements to WantlistConfig that capture the core elements of
the wantlist

```

message WantlistConfig {
    ...

    repeated StoredWantlist = 1;
}

message StoreWantlist {
    string name = 1;
    int64 start_date = 2;
    int64 end_date = 3;
    WantlistType type = 4;
    repeated StoredWantlistEntry = 5;
}

message StoredWantlistEntry {
    int64 id = 1;
    int64 master_id = 2
    int32 index = 3;
}
```

We then hard merge this into the existing wantlist, and store if we made any changes.
The merge is somewhat simple and just validates that (a) index and id / master_id matches
and that (b) we don't have any hanging entries. We save the wantlist out if we've changed.

This keeps everything in sync with a slight impact on current wantlist handling. We also
deprecate the UpdateWantlist RPC call since we can now rebuild from the config.

## Tasks

1. Add proto elements to support wantlist in config
1. Remove UpdateWantlist API Endpoint
1. Add logic to build / merge wantlist on config reload
1. Validate with simple wantlist
