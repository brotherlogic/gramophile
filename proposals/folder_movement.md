<!-----

Yay, no errors, warnings, or alerts!

Conversion time: 0.641 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β34
* Tue Aug 08 2023 17:02:05 GMT-0700 (PDT)
* Source doc: Gramophile Proposal: Folder Movement
----->



# Gramophile Proposal: Folder Movement

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
</p>


<p style="text-align: right">
Draft</p>



### Abstract

Enables gramophile to move records between folders under specified conditions.


### Process

Config should support conditional rules to move records between folders. Such conditional rules may be related to records being listened to and scored, or being of a certain age, or a certain time since the last listen event.


### Field Requirements

We need to add two fields:



* LastListen - string 
    * date representation of the last time the record was listened to
* Goal Folder
    * string name of the folder that this record should be placed into eventually
    * Folder set sync will be enqueued on this basis


### Execution Location

Rules will be evaluated within the collection sync loop post sync and any post intent run. The rule will evaluate the folder that the record should be placed into (default is no move if no rules apply), a folder move will enqueue an intent to move the record and any extra adjustments (e.g. nulling score, weight, width etc.).

No effort will be made to ensure that users can’t create loops - however a record moving more than 100 times a day will be blocked and an issue raised on gram state


### Config requirements

User:

	Add issue proto to capture loop issue

Move:

	Conditional

		Add in field values as necessary

	Destination: String (where GOAL is used to move to the goal folder)

	Adjust:

		Null_score

		Null_cleaned

		Null_wdith

		Null_weight

ApplyMove -> for each rule, apply conditional - > if yes then create intent to change values and change folder

Discogs

Needs to support:

	Folder Moves

	Folder Creation

	Folder Deletion

Config Validation



* Creates and updates folders (folders with no records that do not appear in rules will be deleted)
* Folders that do not appear in rules, but have records will be added as a user issue

Count record moves as part of user config.


### Milestones:



* Discogs support folder moves
* Discogs support folder creation
* Discogs support folder deletion
* Gram config supports “listen”, optional score, with field checks
* Gram config support goal folder, with field checks and recommendations
    * I.e. goal folder is drop down, this should be synced on gram state
* Gram config supports folder creation
* RC sync of last listen time
* RC sync of goal folders
* Define move conditional proto (rule should support name and blocked)
* Add method to apply move on condition
* Rule runs post sync and post intent change, enqueues as necessary
* Create change metadata (record record, rule applied, outcome)
* Move recorded in change metadata
* Validation step on excessive moves, blocks rule, pending update