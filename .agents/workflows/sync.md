---
description: Sync the local main branch with origin/main and compress the session context.
---

1.  **Checkout Main**:
    - Ensure you are on the `main` branch:
      ```bash
      git checkout main
      ```

2.  **Sync with Origin**:
    - Fetch and merge the latest changes from the remote `main` branch:
      ```bash
      git fetch origin main && git merge origin/main
      ```

3.  **Compress Context**:
    - Provide a high-signal summary of the current workspace state, including:
        - The current branch and its sync status.
        - Any recent major changes or completed tasks.
        - A brief overview of pending work or identified issues.
    - This summary serves to "compress" the recent history into a single concise update, allowing the session to remain efficient.
