#!/bin/bash
gh issue create --title "[Frontend] v1 Feature: User Onboarding Flow TUI" --body "Implement the TUI for the User Onboarding Flow for the v1 release. 

Requirements:
- TUI acts as main interface.
- Interactive login mirroring \`gram login\`.
- Loading screen blocking user during initial sync.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Backend] v1 Feature: User Onboarding Flow API" --body "Implement the backend API support for the User Onboarding Flow for the v1 release. 

Requirements:
- Trigger initial collection sync automatically after authentication.
- Expose sync progress/status to the TUI.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Frontend] v1 Feature: Organization Configuration TUI" --body "Implement the TUI for the Organization Configuration Mechanism.

Requirements:
- TUI as primary interface to define shelves/boxes.
- Utilize existing protobuf definitions for config.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Backend] v1 Feature: Organization Configuration API" --body "Implement the backend API support for the Organization Configuration Mechanism.

Requirements:
- Support TUI interfacing with \`gram config\` API to apply changes.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Frontend] v1 Feature: Printing Organizations TUI" --body "Implement the TUI output format for printing organizations.

Requirements:
- Print organization details out to the TUI (similar to existing CLI).

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Backend] v1 Feature: Printing Organizations API" --body "Ensure backend API supports retrieving organization details for the TUI printing feature.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Frontend] v1 Feature: Locating Records TUI" --body "Implement the mechanism for locating records within the TUI.

Requirements:
- Replicate the existing \`locate\` method from the CLI within the TUI.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Backend] v1 Feature: Locating Records API" --body "Ensure backend API supports locating records functionality for the TUI.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Frontend] v1 Feature: Super User Segmentation TUI" --body "Implement Super User Segmentation in the TUI.

Requirements:
- Check user permission level from the protobuf.
- Unhide/enable non-organization features (like sales and wantlists) for authorized super users.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"

gh issue create --title "[Backend] v1 Feature: Super User Segmentation API" --body "Implement backend support for Super User Segmentation.

Requirements:
- Ensure the user's permission level is properly populated in the User protobuf definition and sent to the client.

See requirements: https://github.com/brotherlogic/gramophile/blob/main/v1/requirements.md" --label "seraphine-needs-requirements"
