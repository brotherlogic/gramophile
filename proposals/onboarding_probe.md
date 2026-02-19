# Onboarding Probe

Describes how we can run a probe to (a) onboard a user and then (b)
pull in their collection and (c) set them to live once pulled.

## Process

Since we can't run the actual onboarding process we'll have to simulate the login steps
and then run the background processing and upcycling.

1. Onboard user and store user data in kubernetes secret against prober task
    1. User should have enough collection to require pagination - maybe at least the first 26 records in the db
1. Probe run:
    1. Delete user data (except for the user themselves)
        1. Basically everything under gramophile/user/userid
    1. Mark user as UNKNOWN
    1. Run final onboard step
    1. We should see user go through REFRESHING
    1. Eventually (say after 5 minutes) we should see them IN_WAITLIST

So the prober task may have to insert queue elemets to simulate the first three steps
but we should expect that the prober is able to complete the final step

Since we might run multiple probes against the same user we should lock in the prober task
to verify.

Prober run failure raises a ticket describing the run and associated failure. Logs should
be stored long term in Loki for debugging.
