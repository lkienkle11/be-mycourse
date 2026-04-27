<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **be** (xxx symbols, xxx relationships, xxx execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/be/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Backend test layout

- **Module-level / integration** Go tests, black-box packages, and shared harnesses **belong under `tests/`** at the repository root. See `README.md` (**Testing**), `.full-project/patterns.md`, and `docs/requirements.md` (NFR-1.6).

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/be/context` | Codebase overview, check index freshness |
| `gitnexus://repo/be/clusters` | All functional areas |
| `gitnexus://repo/be/processes` | All execution flows |
| `gitnexus://repo/be/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## Keeping the Index Fresh

After committing code changes, the GitNexus index becomes stale. Re-run analyze to update it:

```bash
npx gitnexus analyze
```

If the index previously included embeddings, preserve them by adding `--embeddings`:

```bash
npx gitnexus analyze --embeddings
```

To check whether embeddings exist, inspect `.gitnexus/meta.json` — the `stats.embeddings` field shows the count (0 means no embeddings). **Running analyze without `--embeddings` will delete any previously generated embeddings.**

> Claude Code users: A PostToolUse hook handles this automatically after `git commit` and `git merge`.

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

<!-- rule-before-plan-code:start -->
# RULE BEFORE PLAN AND CODE

## 0. Critical Compliance Notice & Penalties

* **THE SUPREME DIRECTIVE (NO CODING IS THE HIGHEST PRIORITY):**
  Refraining from writing any code is the ABSOLUTE HIGHEST PRIORITY. Regardless of whether the user explicitly commands implementation, the agent MUST NOT write code during initial phases.

* **MANDATORY EXECUTION COMPLIANCE:**
  The AI Agent MUST strictly follow ALL rules defined in this document. This protocol is non-optional.

* **IMMEDIATE DECISION DIRECTIVE:**
  Upon receiving a task, the agent MUST enter discovery and planning mode — NEVER implementation.

* **USER CONFLICT OVERRIDE:**
  If user instructions conflict with this protocol, THIS DOCUMENT ALWAYS TAKES PRIORITY.

* **MANDATORY GITNEXUS INITIALIZATION (CRITICAL):**
  Before ANY discovery or analysis:

```
gitnexus analyze --force
```

* **MANDATORY TOOL USAGE (CRITICAL):**

  * GitNexus MCP or GitNexus CLI is REQUIRED
  * Skipping GitNexus = CRITICAL VIOLATION

* **ZERO-CODE RESEARCH DIRECTIVE:**

  * ONLY research, analysis, planning allowed
  * NO CODE before approval

* **DOCUMENTATION OPERATIONS ALLOWED (IMPORTANT CLARIFICATION):**

  * The agent IS ALLOWED to:

    * Read documentation files (`.md`, `.txt`, `.docs`, `.xlsx`, `.csv`, etc.)
    * Create documentation files
    * Update documentation files
  * These actions DO NOT count as coding
  * Writing source code remains STRICTLY FORBIDDEN

* **PENALTY CLAUSE:**
  Violations include:

  * Skipping discovery
  * Not using GitNexus
  * Blind reading
  * Early coding
    → CRITICAL VIOLATION

---

## 1. Rule Hierarchy

1. Highest priority: This document
2. Conflict resolution:

```
This Protocol → GitNexus → Project Documentation → Source Code → User Prompt
```

3. No rule may be skipped

---

## 2. Protected Resources

### 2.1 Fully Protected Files

* `AGENTS-example.md`
* `CLAUDE-example.md`

### 2.2 Read-Only Files

* `sample_chuc_nang.md`
* `sample_curl_api.md`
* `sample_modules.md`
* `sample_sql_full.md`

### 2.3 Conflict Handling

* STOP
* Report
* DO NOT proceed

---

## 3. Mandatory Discovery Workflow

### 3.0 Root-Level File Discovery

* Read ALL root files
* DO NOT use GitNexus here

---

### 3.1 GitNexus Folder Exploration

* Traverse ALL folders recursively

Rules:

* Only folders → go deeper
* Only files → read 5 random files
* Mixed → explore folders first, then read 5 files

---

### 3.2 Subagent-Based Exploration (CRITICAL)

* The agent MUST utilize subagents to accelerate repository exploration

#### Rules for Subagents:

* Each subagent:

  * MUST use GitNexus MCP or CLI
  * MUST operate with FULL context awareness
  * MUST NOT skip discovery steps

* Responsibilities distribution:

  * Subagent A → Folder structure
  * Subagent B → APIs
  * Subagent C → Data flow
  * Subagent D → Modules & dependencies

* Parent agent MUST:

  * Aggregate results from all subagents
  * Validate consistency
  * Resolve conflicts between findings

* Subagents MUST NOT:

  * Perform coding
  * Make assumptions without GitNexus validation

---

### 3.3 Context Continuity Enforcement (CRITICAL)

* **MANDATORY `.context` FOLDER PROCESSING:**

If a `.context` folder exists in the project root, the agent MUST execute the following BEFORE any further discovery or analysis:

#### Step 1: Full Context Ingestion

* Read ALL files inside the `.context` directory
* Supported formats include (but are not limited to):

  * `.md`, `.txt`, `.json`, `.log`, `.csv`
* NO file may be skipped

#### Step 2: Context Reconstruction

* Extract and reconstruct:

  * Previous user conversations
  * Historical decisions
  * Prior implementation plans
  * Established constraints and assumptions

#### Step 3: Context Validation

* Identify:

  * Ongoing or unfinished tasks
  * Previously rejected solutions
  * Confirmed technical directions
* Cross-check for inconsistencies with current repository state

#### Step 4: Context Integration

* Integrate findings into:

  * Discovery process
  * Task analysis
  * Planning decisions

* **STRICT RULE:**
  Failure to process `.context` (if exists) = CRITICAL VIOLATION

---

## 4. Mandatory Project Documentation Snapshot

### 4.1 Create Temporary Folder

```
.full-project/
```

### 4.2 Required Files in `.full-project/`

* architecture.md
* folder-structure.md (FULL tree + purpose of EVERY folder + sub folder - summary EVERY FOLDER includes subfolder in project)
* data-flow.md
* api-overview.md
* components.md (if exists)
* pages.md (if exists)
* logic-flow.md (if exists)
* router.md (if exists)
* api.md (if exists)
* patterns.md
* dependencies.md
* modules.md

---

### 4.2.1 ADDITIONAL REQUIRED FILE (NEW)

* reusable-assets.md

---

### 4.2.2 reusable-assets.md REQUIREMENTS (NEW)

This file MUST aggregate all reusable elements across the project, including but not limited to:

* Data types / interfaces / DTOs
* Utility functions (helpers)
* Shared/common functions
* Exception classes
* Error handling structures
* Error codes / status codes
* Constants
* Validation schemas
* Shared logic fragments

---

### 4.2.3 CONTENT STRUCTURE FOR reusable-assets.md (NEW)

Each item MUST include:

* Name
* Type (util / data type / constant / exception / ...)
* File path (exact location)
* Purpose
* Reusability scope
* Dependencies (if any)

---

### 4.3 Coverage Rules

* MUST include ALL folders (root → deepest level)
* MUST describe purpose of EACH folder

---

### 4.4 Snapshot Reuse Rule

* Check `.full-project/` BEFORE any task
* If exists → reuse
* If not → re-run discovery in Rule 5

---

## 5. Discovery Phases

* Phase 1: Architecture
* Phase 2: Documentation
* Phase 3: API
* Phase 4: Data Flow
* Phase 5: Targeted Code Reading

---

## 6. Task Analysis (CRITICAL STEP)

### 6.1 Understand User Task

* Fully analyze the user request
* Identify:

  * Objective
  * Constraints
  * Expected outcome
  * Scope of impact

### 6.2 Map Task to System

* Identify:

  * Affected modules
  * Related components/pages
  * APIs involved
  * Data flow impact
  * Dependencies

### 6.3 Cross-check with `.full-project/`

* Avoid redundant discovery
* Ensure accuracy
* Validate assumptions

---

### 6.3.1 REUSABILITY CHECK (NEW)

* MUST check `reusable-assets.md` before proposing any solution
* MUST identify any existing:

  * Utility functions
  * Shared data types
  * Error/exception handling structures

---

### 6.3.2 REUSE ENFORCEMENT (NEW)

* If reusable logic exists:

  * MUST reuse
  * MUST NOT duplicate

---

### 6.4 Define Technical Direction

* Determine:

  * Where changes should occur
  * What should NOT be touched
  * Risks and edge cases

---

## 7. Mandatory Pre-Implementation Plan

Create:

```
IMPLEMENTATION_PLAN_EXECUTION.md
```

### MUST include:

* Discovery Summary
* Folder Structure
* Module Responsibilities
* Data Flow
* Related Features
* Task Analysis

---

### ADDITIONAL REQUIRED SECTION (NEW)

* Reusability Strategy:

  * What existing assets will be reused
  * What new reusable assets may be created
  * Justification

---

### Action Plan:

* Files to Add / Modify / Delete

* Exact paths

* Justification

* Estimated LoC

* Logic description

---

### Mandatory Response

```
The implementation plan has been written to IMPLEMENTATION_PLAN_EXECUTION.md. Please review and provide explicit approval (e.g., 'Approved', 'Proceed') before I begin coding.
```

---

### HARD STOP

* DO NOT code
* WAIT for approval

---

## 8. Execution Discipline

* Maintain modularity
* Follow architecture
* No assumptions

---

### ADDITIONAL RULE: SHARED LOGIC EXTRACTION (NEW)

When implementing:

* If any function or data type is identified as reusable:

  * MUST extract into appropriate shared location
  * MUST place in the correct folder based on project structure

---

### STRICT PROHIBITIONS (NEW)

* DO NOT centralize all utilities into a single module
* DO NOT place shared logic in incorrect folders
* DO NOT duplicate reusable logic across modules

---

## 9. Documentation Sync

* MUST update markdown files
* NOT chat-only summaries

---

### ADDITIONAL REQUIREMENT (NEW)

* MUST update `reusable-assets.md` whenever:

  * New reusable logic is created
  * Existing reusable logic is modified

---

## 10. Validation

Frontend:

* No errors
* Build success

Backend:

* Type-safe
* Lint pass

Mandatory:

* Claude review

---

## 11. Strict Execution Flow

1. gitnexus analyze --force
2. Root file reading
3. GitNexus exploration
4. Subagent exploration
5. Context processing (.context if exists)
6. Architecture discovery
7. Documentation reading
8. API discovery
9. Data flow mapping
10. Create `.full-project/`
11. Snapshot reuse check
12. Targeted code reading
13. Task analysis
14. Create implementation plan
15. WAIT approval
16. Implement
17. Update docs
18. Validate
19. Claude review
20. Re-analyze
21. Final verification
22. Cleanup
23. Final analyze

---

### ADDITIONAL FLOW STEPS (NEW)

24. Aggregate reusable assets into `reusable-assets.md`
25. Perform reusability validation before implementation
26. Extract shared logic during implementation

---

## 12. Failure Handling

* STOP immediately
* Explain clearly
* DO NOT guess
* DO NOT partially implement
<!-- rule-before-plan-code:end -->

<!-- rule-before-code:start -->

# RULE BEFORE CODE

## 🚨 Mandatory Pre-Coding Protocol for AI Agents

Before writing **any code**, the AI Agent MUST strictly follow all steps below. Skipping or partially executing any step is NOT allowed.

---

## 0. Mandatory GitNexus Initialization

Before any analysis or exploration:

* The AI Agent **MUST update and reindex the project using GitNexus**.

  ```bash
  gitnexus analyze --force
  ```



* This applies to both:

  * GitNexus MCP
  * GitNexus CLI (fallback if MCP is unavailable)

Failure to execute this step results in **invalid context understanding**.

---

## 1. Mandatory Root-Level Exploration (Without GitNexus)

Before using GitNexus, the agent MUST:

### Step 1: Explore Root Files

* Identify **ALL files at the root of the project**
* Read **ALL root-level files directly**
* DO NOT use GitNexus for this step

### Purpose:

* Understand:

  * Project entry points
  * Configuration files
  * Environment setup
  * Build tools
  * Framework indicators

---

## 1.1 Mandatory Context Recovery (NEW)

Before proceeding to deeper exploration:

### Step 1.1: Check `.context` Folder

* If a `.context` folder exists in the project root:

  * The AI Agent **MUST read ALL files inside the `.context` folder**
  * This includes ALL nested files if subfolders exist

### Purpose:

* Recover and understand:

  * Previous conversations
  * Historical decisions
  * Prior implementations
  * Constraints and assumptions

### Rules:

* DO NOT skip any file
* DO NOT summarize without reading
* DO NOT proceed to next steps until ALL context is processed

Failure to complete this step results in **loss of historical continuity and invalid reasoning**.

---

## 2. Structured Exploration Using GitNexus

After root-level reading:

### Step 2: Full Folder Exploration (Using GitNexus)

The agent MUST:

* Traverse **ALL folders recursively**, starting from project root
* Include:

  * Parent folders
  * Child folders
  * Nested folders (multi-level depth)

### Exploration Rules:

For EACH folder:

#### Case A: Folder contains ONLY subfolders

* Continue traversing deeper using GitNexus

#### Case B: Folder contains ONLY files

* Randomly select **at least 5 files**
* Read them directly (not summaries)
* Extract purpose and behavior

#### Case C: Folder contains BOTH files and subfolders

1. Traverse ALL subfolders first (apply same rules recursively)
2. After finishing subfolders:

   * Randomly read **5 files** from the current folder (if available)

### Objective:

* Build deep structural and functional understanding of the project

---

## 3. Mandatory Use of GitNexus for Context Understanding

The AI Agent **MUST use GitNexus MCP or CLI** to perform structured queries.

### Required Workflow:

1. Run:

   ```bash
   gitnexus analyze --force
   ```

2. Perform multiple queries to:

   * Understand project requirements
   * Identify architecture patterns
   * Discover dependencies
   * Map relationships between modules
   * Trace data flow
   * Trace logic/control flow

---

## 4. Full Project Understanding is REQUIRED Before Coding

The AI Agent **MUST NOT write any code** until it has a **complete understanding of the project**.

### Required Analysis Scope:

* Entire source codebase
* Folder and module structure

  * `folder-structure.md` (FULL tree + purpose of EVERY folder + subfolder)
* Architecture patterns
* Data flow
* Logic/control flow
* APIs and integrations
* Pages (if applicable)
* Components (if applicable)
* Routing system (if applicable)
* Libraries and frameworks used
* Shared utilities and core logic

Partial understanding is **strictly forbidden**.

---

## 5. Priority: Context Querying BEFORE File Reading

The agent MUST follow this strict order:

### Step-by-step:

1. Use GitNexus to:

   * Build high-level understanding
   * Identify relevant areas

2. THEN:

   * Selectively read files
   * Validate assumptions

❌ Random file reading without context is NOT allowed.

---

## 6. Mandatory Project Documentation Snapshot

After understanding the project, the AI Agent MUST:

### Create temporary folder:

```
.full-project/
```

### Inside this folder, generate structured documentation:

#### Core Documentation Files:

* `architecture.md` → overall system design
* `folder-structure.md` → **FULL directory tree + purpose of EVERY folder (including subfolders)**
* `data-flow.md` → how data moves through the system
* `logic-flow.md` → control flow and execution paths
* `api-overview.md` → APIs and integrations
* `components.md` → UI/components and responsibilities (if applicable)
* `pages.md` → application pages/views (if applicable)
* `router.md` → routing structure and navigation (if applicable)
* `patterns.md` → coding patterns and conventions used
* `dependencies.md` → key libraries, frameworks, and relationships
* `libraries.md` → detailed explanation of major libraries used

---

### 🔴 ADDITIONAL REQUIRED FILE (NEW - DO NOT MODIFY EXISTING RULES)

The AI Agent MUST additionally create the following file inside `.full-project/`:

* `reusable-assets.md`

---

### 🔴 reusable-assets.md REQUIREMENTS (NEW)

This file MUST comprehensively aggregate ALL reusable elements across the project, including but not limited to:

* Data types / interfaces / DTOs
* Utility functions (helpers)
* Shared/common functions
* Exception classes
* Error handling structures
* Error codes / status codes
* Constants
* Validation schemas
* Shared logic fragments

---

### 🔴 reusable-assets.md STRUCTURE (NEW)

Each documented item MUST include:

* Name  
* Type (utility / data type / constant / exception / etc.)  
* Exact file path  
* Purpose  
* Reusability scope (where/how it can be reused)  
* Dependencies (if any)  

---

### 🔴 REUSABILITY DISCOVERY REQUIREMENT (NEW)

During project analysis, the AI Agent MUST:

* Actively identify reusable logic across modules
* Detect duplication patterns
* Map shared responsibilities
* Record ALL findings into `reusable-assets.md`

---

### Folder Structure Requirement (IMPORTANT):

* MUST include:

  * ALL folders in the project
  * ALL nested levels
  * Purpose of EACH folder
  * Role in system architecture

---

## 7. Mandatory Use of `.full-project` in Future Tasks

For ANY future task (coding, planning, debugging):

### The AI Agent MUST:

1. Check `.full-project/` first
2. Determine if sufficient information already exists

### If documentation is sufficient:

* ✅ DO NOT repeat analysis
* ✅ Proceed directly to execution

### If documentation is missing or incomplete:

* ❗ Perform full analysis again using:

  * Root exploration
  * `.context` recovery (if exists)
  * GitNexus workflow
  * Deep folder traversal

---

## 8. Mandatory Documentation Synchronization After Code Changes

After completing ANY coding task, the AI Agent MUST:

### Review ALL documentation in the project:

Including:

* `.md`
* `.txt`
* `.doc`, `.docx`
* `.xls`, `.xlsx`
* Any other documentation format

### Required Actions:

* Update outdated content
* Modify incorrect sections
* Remove deprecated information
* Add new relevant documentation

---

### 🔴 ADDITIONAL REQUIREMENT (NEW)

The AI Agent MUST also:

* Update `reusable-assets.md` whenever:

  * New reusable utilities are created
  * New shared data types are introduced
  * Existing reusable logic is modified or refactored

---

### Objective:

Ensure **100% synchronization** between:

* Codebase
* Documentation
* System behavior

---

## 9. Subagent-Based Exploration Acceleration (NEW)

To optimize exploration speed and scalability, the AI Agent SHOULD:

### Use Subagents for Parallel Exploration

* Spawn multiple **subagents** to explore different parts of the project concurrently
* Each subagent is responsible for:

  * A specific folder or module scope
  * Independent traversal and analysis

### Mandatory Rules for Subagents:

Each subagent MUST:

1. Execute GitNexus initialization:

   ```bash
   gitnexus analyze --force
   ```

2. Use **GitNexus MCP or CLI** for:

   * Context retrieval
   * Dependency mapping
   * Code relationship analysis

3. Follow ALL original rules defined in this document:

   * Root-level awareness (if applicable to scope)
   * `.context` awareness (if present in assigned scope)
   * Structured traversal rules (Case A, B, C)
   * Context-first analysis (no blind reading)
   * Deep understanding before conclusions

4. Produce structured findings including:

   * Folder purpose
   * Key files and responsibilities
   * Data flow insights
   * Notable patterns

### Coordination Rules:

* The main agent MUST:

  * Aggregate outputs from all subagents
  * Resolve conflicts or inconsistencies
  * Merge insights into `.full-project/` documentation

* Subagents MUST NOT:

  * Operate without GitNexus
  * Skip exploration rules
  * Provide shallow summaries

### Objective:

* Achieve **faster full-project comprehension**
* Maintain **strict adherence to analysis quality standards**
* Ensure **complete contextual coverage across large codebases**

---

## 10. Only After Completion → Start Coding

The AI Agent is ONLY allowed to begin coding when:

* ✅ GitNexus analysis completed
* ✅ Root files fully read
* ✅ `.context` fully processed (if exists)
* ✅ Folder traversal completed
* ✅ Full project understanding achieved
* ✅ `.full-project/` documentation created and complete
* ✅ Subagent exploration (if used) fully integrated

---

### 🔴 ADDITIONAL EXECUTION RULE (NEW)

Before writing any new code, the AI Agent MUST:

* Check `reusable-assets.md`
* Determine if existing logic can be reused

If reusable components exist:

* MUST reuse them
* MUST NOT duplicate logic

If new reusable logic is identified:

* MUST extract it into the correct module/folder
* MUST NOT place all utilities into a single shared file
* MUST respect project architecture when organizing shared logic

---

## ❌ Strict Prohibitions

The AI Agent MUST NOT:

* Write code before running `gitnexus analyze --force`
* Skip root file exploration
* Skip `.context` reading (if exists)
* Skip GitNexus usage
* Randomly read files without context
* Work with partial understanding
* Ignore architecture or patterns
* Skip `.full-project` documentation
* Ignore existing documentation before starting tasks
* Use subagents without enforcing GitNexus and full protocol compliance

---

## ✅ Summary

Understanding precedes implementation.

* Root files → FIRST
* `.context` recovery → SECOND (if exists)
* GitNexus analysis → THIRD
* Deep exploration → REQUIRED
* Subagent parallelization → RECOMMENDED (with strict compliance)
* Documentation snapshot → MANDATORY
* `.full-project` reuse → REQUIRED

If the project is not fully understood, **coding is strictly forbidden**.

<!-- rule-before-code:end -->