# Config

Config selection allows us to control who has access to what in gramophile
and ensures that we can work on new features without affecting other users.

## Control

Users interact with gramophile by setting their config, this config tells
gramohpile what it should do with regard to thei records and such. We need
to have a way to allow some users to access new config settings whilst
ensuring that they are not available to all users.

To do this we have a concept of resetting config values - i.e. here we
evaluate a proto path against the config e.g. config_settings.config_value
and grouping users into three groups:

1. USER_STANDARD
1. USER_BETA
1. USER_OMNIPOTENT

We then set a config mapping on those users (in code). Where we say

```plaintext
USER_BETA -> [config_settings.config_value, ...]
USER_STANDARD -> [config_setings.config_value, ...]
```

On each config load we do a post process of applying these config settings,
where set values are overwritten with theie defaults (FALSE for booleans, the
zeroth element for an enum).

That way we can slowly make features available to (a) a beta population
and (b) the general population at large.

## User control

Users can self select into the BETA group at any time, a setting in their
config. Only hard coded users can be OMNIPOTENT.

## Tasks

1. Support user type configuration
1. Enforce omnipotent ruling on user type change
1. Set proto to support user type settings
1. Allow for config setting changes in setconfig
1. Build a set of rules for beta users
1. Add test user, valudate config adjustment occurs
