# RULE BEFORE CODE

## 🚨 Mandatory Pre-Coding Protocol for AI Agents

Before writing **any code**, the AI Agent MUST strictly follow all steps below. Skipping or partially executing any step is NOT allowed.

---

## 0. Mandatory GitNexus Initialization

Before any analysis or exploration:

- The AI Agent **MUST update and reindex the project using GitNexus**.

  ```bash
  gitnexus analyze --force
````

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
* Folder and module structure - folder-structure.md (FULL tree + purpose of EVERY folder + sub folder - summary EVERY FOLDER includes subfolder in project)  
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
* ✅ Folder traversal completed
* ✅ Full project understanding achieved
* ✅ `.full-project/` documentation created and complete
* ✅ Subagent exploration (if used) fully integrated

---

## ❌ Strict Prohibitions

The AI Agent MUST NOT:

* Write code before running `gitnexus analyze --force`
* Skip root file exploration
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
* GitNexus analysis → SECOND
* Deep exploration → REQUIRED
* Subagent parallelization → RECOMMENDED (with strict compliance)
* Documentation snapshot → MANDATORY
* `.full-project` reuse → REQUIRED

If the project is not fully understood, **coding is strictly forbidden**.

