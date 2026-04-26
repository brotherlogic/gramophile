---
description: Automatically create a feature branch (if on main), commit with a generated message, and push to GitHub.
---

1.  **Check Current Branch**:
    ```bash
    git branch --show-current
    ```

2.  **Determine Action**:
    - If the output is `main`:
        - Generate a descriptive feature branch name based on your recent changes (e.g., `feature/add-docker-registry`).
        - Create and switch to the new branch:
          ```bash
          git checkout -b feature/<your-descriptive-name>
          ```
    - If the output is NOT `main`:
        - Proceed on the current branch.

3.  **Stage Changes**:
    - Stage all modified and new files:
      ```bash
      git add .
      ```

4.  **Generate Commit Message**:
    - Create a concise and descriptive commit message that summarizes the work done (e.g., "Add GitHub workflow for Docker publishing and auto-tagging"). If we're on a bug branch ('bug/xxxxx'), the commit message should end with "This closes #xxxx" where the number is the bug number.
    - Commit the changes:
      ```bash
      git commit -m "<your-generated-message>"
      ```

5.  **Push to GitHub**:
    - Push the branch to the remote repository:
      ```bash
      git push origin $(git branch --show-current)
      ```

6.  **Locate Pull Request**:
    - The Pull Request is created automatically upon pushing to a feature branch. Find the Pull Request associated with the newly pushed branch using the `gh` tool. This may require retries if the PR is still being processed.
      ```bash
      gh pr list --head $(git branch --show-current)
      ```

7.  **Trigger Review**:
    - Initiate an AI review by posting a comment on the Pull Request:
      ```bash
      gh pr comment $(git branch --show-current) --body "/gemini-review"
      ```

8.  **Track and Address Review**:
    - Monitor the Pull Request for feedback. You must actively fetch and read review comments once they are available.
      ```bash
      gh pr view $(git branch --show-current) --comments
      ```
    - Analyze the feedback and make necessary adjustments to the code to address the suggestions or requirements.

9.  **Push Adjustments**:
    - Stage, commit, and push the updates to the same branch:
      ```bash
      git add .
      git commit -m "Address review feedback"
      git push origin $(git branch --show-current)
      ```
    - Repeat steps 8 and 9 until the AI review is satisfied.

10. **Human Review**:
    - Once the AI review is satisfied, assign brotherlogic for the final human review:
      ```bash
      gh pr edit $(git branch --show-current) --add-reviewer brotherlogic
      ```