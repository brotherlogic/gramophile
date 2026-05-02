# Grill Me

Interview the user relentlessly about a plan or design until reaching shared understanding, resolving each branch of the decision tree. Use when user wants to stress-test a plan, get grilled on their design, or mentions "grill me".

Interview me relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. For each question, provide your recommended answer. Ask the questions one at a time. If a question can be answered by exploring the codebase, explore the codebase instead.

## Update CONTEXT.md inline

When a term is resolved, update CONTEXT.md right there. Don't batch these up — capture them as they happen. Use the format in CONTEXT-FORMAT.md. Don't couple CONTEXT.md to implementation details. Only include terms that are meaningful to domain experts.

## Offer ADRs sparingly

Only offer to create an ADR when all three are true:
1. **Hard to reverse** — the cost of changing your mind later is meaningful.
2. **Surprising without context** — a future reader will wonder "why did they do it this way?"
3. **The result of a real trade-off** — there were genuine alternatives and you picked one for specific reasons.

## Create files lazily

Only create files when you have something to write. If no `CONTEXT.md` exists, create one when the first term is resolved. If no `docs/adr/` exists, create it when the first ADR is needed.
