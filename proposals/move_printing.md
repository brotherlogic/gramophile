<!-----

Conversion time: 0.269 seconds.

Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β36
* Sun Jun 23 2024 15:25:19 GMT-0700 (PDT)
* Source doc: Gramophile Move Printing
* Tables are currently converted to HTML tables.
----->

# Gramophile Move Printing

<p style="text-align: right">
Brotherlogic</p>

<p style="text-align: right">
Draft</p>

<p style="text-align: right">
</p>

### Abstract

This document describes a process by which gramophile can print moves when they are made. This should be somewhat agnostic to where the move originates (i.e. whether it comes from gramohpile or another orchestrator).

### Process

In the update loop if we see that a record has moved (i.e. it’s folder id is different to the stored value), then we should create a new object in the queue which is a Move. The move object captures the context between the original location and the new location - specifically it captures the position of the record in the original location and the position of the record in the new location. Whilst we limit it to folder id initially, we can consider moving over to allow moves within location at some point.

Moves are stored and periodically printed (initially every 5 minutes) - we control how the print is actuated through config settings that control how much context is added:

```
<ID Number>
<Original Location> <Shelf / Slot>
<Before Context>
<Artist> - <Title>
<After Context>

-> <New Location> <Shelf / Slot>
<Before Context>
<Artist> - <Title>
<After Context>


Where Contexts are the surrounding records up to a max of 10
```

This allows us to both locate the record to be moved and to situate it in its new home. A move is considered complete when it has both the original location and the new location. We also support setting some locations as non-printable (i.e. they are a temporary holding place), in which case they are skipped in the move setting. Once the move is printed, we delete the move.

We use the [github.com/brotherlogic/printbridge](github.com/brotherlogic/printbridge) to support printing.

To support this we have the new settings:

```
PrintSettings {
   Context: int32 # The number of surrounding records
   PrintTarget: The dial location for printing
}

Move {
  Timestamp: int64
  Id: int64
  Record_iid: int64
  Origin: Location
  Destination: Location
}

Location {
  String location_name
  Repeated Context Before
  Repeated Context After
}

Context {
  Int32 index
  Int64 iid
}
```

### Milestones

1. Add this doc into the repo
2. Set the config settings for handling move printing
3. Support the save and load and listing of moves
4. In the update loop detect a folder a change and create a move and save it
5. Add a loop processor into validator
6. Loop processor should find moves to printed and then build the print represetnation
7. Prints shuold be shipped off to printbridge, delete on success
8. Grafana dashbaord for tracking the size of the print queue
