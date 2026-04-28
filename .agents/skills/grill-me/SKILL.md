---
name: grill-me
description: Interview the user relentlessly about a plan or design until reaching shared understanding, resolving each branch of the decision tree.
---

# Grill Me

Interview the user relentlessly about a plan or design until reaching shared understanding, resolving each branch of the decision tree.

## Usage

Use this skill when the user wants to stress-test a plan, get grilled on their design, or explicitly mentions "grill me".

## Instructions

Interview me relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one.

### Workflow

1. **Read the plan**: Understand what the user has described so far.
2. **Identify the decision tree**: Map out every branch (e.g., architecture, data model, UX, edge cases, deployment, dependencies).
3. **Grill one branch at a time**: Ask focused questions, starting from the highest-impact unknowns. Don't move on until the branch is resolved.
4. **Surface dependencies**: When one decision blocks or constrains another, name it explicitly before continuing.
5. **Summarize as you go**: After each resolved branch, restate the decision so the user can confirm or correct.
6. **Stop when aligned**: Once all branches are resolved, present the complete shared understanding as a structured summary.

### Rules

- **Never assume**: If something is ambiguous, ask.
- **One topic at a time**: Don't bundle unrelated questions.
- **Provide recommendations**: For each question, provide your recommended answer based on best practices and the current codebase.
- **Explore first**: If a question can be answered by exploring the codebase, explore the codebase instead of asking.
- **Push back**: If a decision seems risky, contradictory, or lacks a measurable success criterion, say so.
- **No implementation**: This skill is for planning only. Do not write implementation code.
- **Be direct**: Skip pleasantries. Get straight to the high-signal questions.
- **Track progress**: Keep a mental map of resolved vs. open branches so the user knows how much is left.
