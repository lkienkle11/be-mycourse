# Enhanced Instruction Set for AI Agent

## 1. Prioritize GitNexus for Context Understanding

Always prioritize using GitNexus (via MCP or CLI) to gain the deepest
possible understanding of the project context **before reading any
source code files directly**.\
This ensures a structured, high-level overview and avoids fragmented or
misleading interpretations from isolated files.

## 2. Leverage Project Documentation

Combine GitNexus insights with project documentation: - Use GitNexus
tools or manual search to locate all relevant documentation (README,
design docs, architecture notes, etc.). - Ensure a comprehensive
understanding of system design, conventions, and dependencies before
making changes.

## 3. Continuous Documentation Updates

Throughout the entire workflow (before, during, and after
implementation): - Continuously update or create documentation where
necessary. - Keep documentation aligned with actual code changes. -
Regularly refresh GitNexus indexes and analysis using:

    analyze --force

-   Ensure all newly added knowledge is indexed and discoverable.

## 4. Frontend (FE) Validation Requirements

When working on frontend tasks: - Thoroughly check for type errors
(e.g., TypeScript issues). - Ensure the project builds successfully
without errors. - Run formatting tools consistent with the project's
setup (e.g., Prettier, ESLint, or IDE-integrated formatters). - Validate
UI behavior if applicable (basic sanity checks).

## 5. Backend (BE) Validation Requirements

When working on backend tasks: - Verify type correctness (if using typed
languages). - Ensure the build process completes successfully. - Run
relevant linters or static analysis tools if available. - Confirm no
runtime-breaking issues are introduced (at least via basic checks or
tests if present).

## 6. General Engineering Discipline

-   Do not make assumptions when context is unclear---resolve ambiguity
    using GitNexus or documentation first.
-   Prefer consistency with existing architecture and coding patterns.
-   Minimize unnecessary changes outside the scope of the task.
-   Ensure all changes are traceable, documented, and justifiable.
