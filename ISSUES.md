# Notes Management System - GitHub Issue Processing Workflow

This document serves as the entry point and index for Seraphine's issue-processing workflows. It outlines the general rules and lists the specific workflow files for each stage in the issue lifecycle.

---

## 🚫 Critical General Rules
1. **Scope Adherence**: The agent should only address the labeled issue, and it must stop once the issue is unlabeled.
2. **Termination Rule**: **The agent should not proceed to the next label.** Once you have removed a label from the bug (or a PR is merged), you should stop execution immediately. Do not trigger or begin processing the next stage or label in the same run.

---

## 🏷️ Workflow Stages & Labels

When an issue is labeled, refer to the corresponding workflow document under `.agents/workflows/` for detailed step-by-step instructions:

1. **Requirements gathering**
   - **Label**: `seraphine-needs-requirements` (or variant `seraphine-need-requirements`)
   - **Workflow Guideline**: [seraphine-needs-requirements.md](file:///workspaces/seraphine/.agents/workflows/seraphine-needs-requirements.md)

2. **Technical implementation plan formulation**
   - **Label**: `seraphine-needs-implementation-plan`
   - **Workflow Guideline**: [seraphine-needs-implementation-plan.md](file:///workspaces/seraphine/.agents/workflows/seraphine-needs-implementation-plan.md)

3. **Issue breakdown**
   - **Label**: `seraphine-break-down-issue`
   - **Workflow Guideline**: [seraphine-break-down-issue.md](file:///workspaces/seraphine/.agents/workflows/seraphine-break-down-issue.md)

4. **Component implementation**
   - **Label**: `seraphine-ready-to-implement`
   - **Workflow Guideline**: [seraphine-ready-to-implement.md](file:///workspaces/seraphine/.agents/workflows/seraphine-ready-to-implement.md)

5. **Bug triage and resolution**
   - **Label**: `seraphine-bug`
   - **Workflow Guideline**: [seraphine-bug.md](file:///workspaces/seraphine/.agents/workflows/seraphine-bug.md)

---

## 🛠️ Summary of Expected Label State Transitions

| Phase | Parent Issue Label(s) | Sub-Issue Title & Label(s) |
| :--- | :--- | :--- |
| **Requirements Gathering** | `seraphine-needs-requirements` | *None (Not yet created)* |
| **Requirements Approved** | *(Label Removed)* | `[Implementation Plan] <Title>` labeled with `seraphine-needs-implementation-plan` |
| **Implementation Plan Drafting** | *None* | `[Implementation Plan] <Title>` labeled with `seraphine-needs-implementation-plan` |
| **Implementation Plan Approved** | *None* | **Implementation Plan:** Label removed (remains Open).<br>**Breakdown Sub-Issue:** `[Breakdown] <Title>` labeled with `seraphine-break-down-issue` (created as a native GitHub sub-issue) |
| **Issue Breakdown** | *None* | **Breakdown Issue:** `seraphine-break-down-issue` removed (remains Open).<br>**Child Sub-Issues:** `[Sub-Issue] <Action>` labeled with `seraphine-ready-to-implement` (created as native GitHub sub-issues) |
| **Implementation** | *None* | **Breakdown Issue:** Closed when all child sub-issues are closed (cascading to close Implementation Plan and Parent issues).<br>**Child Sub-Issues:** Labeled with `seraphine-ready-to-implement`. Closed programmatically via PR submission. |
| **Bug Triage (Simple)** | `seraphine-bug` | *None (Direct fix implemented and PR submitted)* |
| **Bug Triage (Complex/Failed)** | `seraphine-bug` (Removed) | New issue labeled with `seraphine-needs-requirements` to initiate requirements gathering |
