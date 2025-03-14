# Manual Wantlist Addition

<p style="text-align: right">
Brotherlogic</p>


<p style="text-align: right">
2023-02-12</p>


<p style="text-align: right">
Draft</p>



### Abstract

In DISCOGS or HYBRID mode a manual want addition (i.e. one made through the site, rather than through gramophile) is retained. This proposal defines what happens to manual wants in GRAMOPHILE managed mode.


### Options

Through a flag setting the user can opt to DROP or TRANSFER a manual want. In the case of DROP, the want is hard dropped - this drop being recorded in the users want history. In the case of TRANSFER the want is moved to a specified list. Wants are added at the end of the list - so in cases where the list is setup to be ONE_BY_ONE or DATE_BOUND the ordering is retained. 

For DATE_BOUND lists this may lead to instability since extending the length of the list may have strange effects on the timing of the want additions. Wants added to the list once the end date has passed will effectively be ignored. 

If a non-existant list is specified as the transfer destination, the list is created as EN_MASSE and can be edited to differ after the fact.

If the


### Configuration



1. Add configuration to WantsConfig
    1. move_manual : WANT_MOVE_UNKNOWN, WANT_MOVE_DROP, WANT_MOVE_TRANSFER
    2. Move_destination: string name of wantlist for destination


### Application



1. In the want sync
    1. If we have a un-credited want (i.e. one not attached to a given list) and we’re in GRAMOPHILE managed mode
        1. DROP
            1. Delete the want
        2. TRANSFER	
            2. Create the wantlist if it doesn’t exist
            3. Add the want to the list


### Milestones



1. This proposal agreed and added
2. Configuration Settings are made
3. Configuration setting with WANT_MOVE_UNKNOWN (i.e. default setting) causes a configuration set error. Thus we have to set WANT_MOVE_DROP or WANT_MOVE_TRANSFER.
4. Want loop processing captures uncredited wants
5. Uncredited wants are handled according the correct scheme.