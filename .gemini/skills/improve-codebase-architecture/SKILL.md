# Improve Codebase Architecture

Find deepening opportunities in a codebase, informed by the domain language in CONTEXT.md and the decisions in docs/adr/. Use when the user wants to improve architecture, find refactoring opportunities, consolidate tightly-coupled modules, or make a codebase more testable and AI-navigable.

## Instructions

1. **Analyze Context**: Read `CONTEXT.md` and `docs/adr/` to understand the established domain language and architectural decisions.
2. **Identify Shallow Modules**: Search for "shallow" modules—files or components where the interface is nearly as complex as the implementation, or where understanding one concept requires jumping between many small, fragmented files.
3. **Find Coupling**: Identify clusters of tightly-coupled modules that lack clear boundaries or "seams."
4. **Propose Deepening**: Suggest "deepening" opportunities where multiple shallow modules can be collapsed into a single deep module with a thin, stable interface and a clear test boundary.
5. **Generate RFCs**: Create GitHub issue RFCs that outline concrete refactoring plans, focusing on vertical slices that improve testability and reduce mental load for both humans and AI.

## Available Resources

- `CONTEXT.md`: The source of truth for domain terminology and system boundaries.
- `docs/adr/`: Architectural Decision Records.
- `LANGUAGE.md`: Precise vocabulary (module, interface, depth, seam, adapter) to be used in all architectural discussions.
