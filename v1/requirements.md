# Gramophile v1 Requirements

## Destination
A finalized, locked-in list of the core features and user journeys necessary to constitute the v1 release of Gramophile.

## Scope and Goals
- **Domain**: Gramophile (Record collection management interfacing with Discogs).
- **Core v1 Focus**: Physical organization of the collection.
- **Constraint**: Tooling should only support functionality that drives the organization goal.
- **Interface**: The primary interface for v1 will be a Terminal User Interface (TUI).

## Key Features & User Stories

### 1. User Onboarding Flow
- **Description**: A seamless onboarding experience for new and returning users.
- **Requirements**:
  - A TUI acts as the main interface.
  - It triggers an interactive login mirroring the `gram login` flow to link the user's Discogs account.
  - After login, it blocks the user with a loading screen displaying sync progress and waitlist promotion until the backend's initial sync completes.
  - Upon completion, the user enters the main application state.

### 2. Organization Configuration Mechanism
- **Description**: Users must be able to define the physical layout of their organization (e.g., shelves, boxes).
- **Requirements**:
  - The TUI serves as the primary interface for users to define their organization.
  - It utilizes the existing protobuf definitions for configuration.
  - The TUI interfaces with the `gram config` command/API to apply these changes.

### 3. Output Format for Printing Organizations
- **Description**: Users need to see the details of their configured organization.
- **Requirements**:
  - For v1, organization details will be printed out to the TUI, similar to how the existing CLI functions.
  - *Note: PDF generation is planned as a fast follow in a future release.*

### 4. Mechanism for Locating Records
- **Description**: The system must support finding specific records within the defined organization (e.g., on a shelf or in a box).
- **Requirements**:
  - Replicate the existing `locate` method from the `gram` CLI.
  - Implement this locate functionality directly within the TUI for v1.

## Super User Segmentation
- **Description**: Cleanly segment active non-organization features (like sales and wantlists) so they are hidden from standard v1 users but available to super users.
- **Requirements**:
  - Leverage the existing feature exposure mechanism in the codebase.
  - The user's permission level, as stated in the User protobuf definition, will dictate access. The TUI will check this level and unhide or enable these features for authorized super users.

## Out of Scope for v1
- General Sales features
- General Wantlist features
