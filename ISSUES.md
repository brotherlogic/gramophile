# Notes Management System - GitHub Issue Processing Workflow

This document serves as the entry point and index for Gramophile's issue-processing workflows. It outlines the general rules and lists the specific workflow files for each stage in the issue lifecycle.

---

## 🚫 Critical General Rules
1. **Scope Adherence**: The agent should only address the labeled issue, and it must stop once the issue is unlabeled.
2. **Termination Rule**: **The agent should not proceed to the next label.** Once you have removed a label from the bug (or a PR is merged), you should stop execution immediately. Do not trigger or begin processing the next stage or label in the same run.

---

## 🏷️ Workflow Stages & Labels

When an issue is labeled, refer to the corresponding workflow document under `.agents/workflows/` for detailed step-by-step instructions:

1. **Requirements gathering**
   - **Label**: `gramophile-needs-requirements` (or variant `gramophile-need-requirements`)
   - **Workflow Guideline**: [gramophile-needs-requirements.md](file:///workspaces/gramophile/.agents/workflows/gramophile-needs-requirements.md)

2. **Technical implementation plan formulation**
   - **Label**: `gramophile-needs-implementation-plan`
   - **Workflow Guideline**: [gramophile-needs-implementation-plan.md](file:///workspaces/gramophile/.agents/workflows/gramophile-needs-implementation-plan.md)

3. **Issue breakdown**
   - **Label**: `gramophile-break-down-issue`
   - **Workflow Guideline**: [gramophile-break-down-issue.md](file:///workspaces/gramophile/.agents/workflows/gramophile-break-down-issue.md)

4. **Component implementation**
   - **Label**: `gramophile-ready-to-implement`
   - **Workflow Guideline**: [gramophile-ready-to-implement.md](file:///workspaces/gramophile/.agents/workflows/gramophile-ready-to-implement.md)

5. **Bug triage and resolution**
   - **Label**: `gramophile-bug`
   - **Workflow Guideline**: [gramophile-bug.md](file:///workspaces/gramophile/.agents/workflows/gramophile-bug.md)

---

## 🛠️ Summary of Expected Label State Transitions

| Phase | Parent Issue Label(s) | Sub-Issue Title & Label(s) |
| :--- | :--- | :--- |
| **Requirements Gathering** | `gramophile-needs-requirements` | *None (Not yet created)* |
| **Requirements Approved** | *(Label Removed)* | `[Implementation Plan] <Title>` labeled with `gramophile-needs-implementation-plan` |
| **Implementation Plan Drafting** | *None* | `[Implementation Plan] <Title>` labeled with `gramophile-needs-implementation-plan` |
| **Implementation Plan Approved** | *None* | **Implementation Plan:** Label removed (remains Open).<br>**Breakdown Sub-Issue:** `[Breakdown] <Title>` labeled with `gramophile-break-down-issue` |
| **Issue Breakdown** | *None* | **Breakdown Issue:** `gramophile-break-down-issue` removed (remains Open).<br>**Child Sub-Issues:** `[Sub-Issue] <Action>` labeled with `gramophile-ready-to-implement` |
| **Implementation** | *None* | **Breakdown Issue:** Closed when all child sub-issues are closed (cascading to close Implementation Plan and Parent issues).<br>**Child Sub-Issues:** Labeled with `gramophile-ready-to-implement`. Closed programmatically via PR submission. |
| **Bug Triage (Simple)** | `gramophile-bug` | *None (Direct fix implemented and PR submitted)* |
| **Bug Triage (Complex/Failed)** | `gramophile-bug` (Removed) | New issue labeled with `gramophile-needs-requirements` to initiate requirements gathering |
