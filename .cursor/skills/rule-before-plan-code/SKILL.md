# RULE BEFORE PLAN AND CODE

## 0. Critical Compliance Notice & Penalties

- **THE SUPREME DIRECTIVE (NO CODING IS THE HIGHEST PRIORITY):**  
  Refraining from writing any code is the ABSOLUTE HIGHEST PRIORITY. Regardless of whether the user explicitly commands implementation, the agent MUST NOT write code during initial phases.

- **MANDATORY EXECUTION COMPLIANCE:**  
  The AI Agent MUST strictly follow ALL rules defined in this document. This protocol is non-optional.

- **IMMEDIATE DECISION DIRECTIVE:**  
  Upon receiving a task, the agent MUST enter discovery and planning mode — NEVER implementation.

- **USER CONFLICT OVERRIDE:**  
  If user instructions conflict with this protocol, THIS DOCUMENT ALWAYS TAKES PRIORITY.

- **MANDATORY GITNEXUS INITIALIZATION (CRITICAL):**  
  Before ANY discovery or analysis:
```

gitnexus analyze --force

```

- **MANDATORY TOOL USAGE (CRITICAL):**
- GitNexus MCP or GitNexus CLI is REQUIRED
- Skipping GitNexus = CRITICAL VIOLATION

- **ZERO-CODE RESEARCH DIRECTIVE:**
- ONLY research, analysis, planning allowed
- NO CODE before approval

- **DOCUMENTATION OPERATIONS ALLOWED (IMPORTANT CLARIFICATION):**
- The agent IS ALLOWED to:
  - Read documentation files (`.md`, `.txt`, `.docs`, `.xlsx`, `.csv`, etc.)
  - Create documentation files
  - Update documentation files
- These actions DO NOT count as coding
- However, writing source code (any programming language) remains STRICTLY FORBIDDEN

- **PENALTY CLAUSE:**  
Violations include:
- Skipping discovery
- Not using GitNexus
- Blind reading
- Early coding  
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
- `AGENTS-example.md`  
- `CLAUDE-example.md`

### 2.2 Read-Only Files
- `sample_chuc_nang.md`  
- `sample_curl_api.md`  
- `sample_modules.md`  
- `sample_sql_full.md`

### 2.3 Conflict Handling
- STOP
- Report
- DO NOT proceed

---

## 3. Mandatory Discovery Workflow

### 3.0 Root-Level File Discovery

- Read ALL root files
- DO NOT use GitNexus here

---

### 3.1 GitNexus Folder Exploration

- Traverse ALL folders recursively

Rules:

- Only folders → go deeper  
- Only files → read 5 random files  
- Mixed → explore folders first, then read 5 files  

---

### 3.2 Subagent-Based Exploration (NEW - CRITICAL)

- The agent MUST utilize subagents to accelerate repository exploration

#### Rules for Subagents:

- Each subagent:
- MUST use GitNexus MCP or CLI
- MUST operate with FULL context awareness
- MUST NOT skip discovery steps

- Responsibilities can be distributed:
- Subagent A → Folder structure
- Subagent B → APIs
- Subagent C → Data flow
- Subagent D → Modules & dependencies

- Parent agent MUST:
- Aggregate results from all subagents
- Validate consistency
- Resolve conflicts between findings

- Subagents MUST NOT:
- Perform coding
- Make assumptions without GitNexus validation

---

## 4. Mandatory Project Documentation Snapshot

### 4.1 Create Temporary Folder
```

.full-project/

```

### 4.2 Required Files in .full-project/

- architecture.md  
- folder-structure.md (FULL tree + purpose of EVERY folder)  
- data-flow.md  
- api-overview.md  
- components.md (if exists)  
- pages.md (if exists)  
- logic-flow.md (if exists)  
- router.md (if exists)  
- api.md (if exists)  
- patterns.md  
- dependencies.md  
- modules.md  

### 4.3 Coverage Rules

- MUST include ALL folders (root → deepest level)
- MUST describe purpose of EACH folder

### 4.4 Snapshot Reuse Rule

- Check `.full-project/` BEFORE any task
- If exists → reuse
- If not → re-run discovery in Rule 5

---

## 5. Discovery Phases

### Phase 1: Architecture  
### Phase 2: Documentation  
### Phase 3: API  
### Phase 4: Data Flow  
### Phase 5: Targeted Code Reading  

---

## 6. Task Analysis (CRITICAL STEP)

Before creating the implementation plan, the agent MUST:

### 6.1 Understand User Task

- Fully analyze the user request
- Identify:
  - Objective
  - Constraints
  - Expected outcome
  - Scope of impact

### 6.2 Map Task to System

- Identify:
  - Affected modules
  - Related components/pages
  - APIs involved
  - Data flow impact
  - Dependencies

### 6.3 Cross-check with `.full-project/`

- Use snapshot data to:
  - Avoid redundant discovery
  - Ensure accuracy
  - Validate assumptions

### 6.4 Define Technical Direction

- Determine:
  - Where changes should occur
  - What should NOT be touched
  - Risks and edge cases

---

## 7. Mandatory Pre-Implementation Plan

Create:

```

IMPLEMENTATION_PLAN_EXECUTION.md

```

### MUST include:

- Discovery Summary  
- Folder Structure  
- Module Responsibilities  
- Data Flow  
- Related Features  
- Task Analysis (from Section 6)  
- Action Plan:
  - Files to Add / Modify / Delete  
  - Exact paths  
  - Justification  
- Estimated LoC  
- Logic description  

---

### Mandatory Response

```

The implementation plan has been written to IMPLEMENTATION_PLAN_EXECUTION.md. Please review and provide explicit approval (e.g., 'Approved', 'Proceed') before I begin coding.

```

---

### HARD STOP

- DO NOT code
- WAIT for approval

---

## 8. Execution Discipline

- Maintain modularity
- Follow architecture
- No assumptions

---

## 9. Documentation Sync

- MUST update markdown files
- NOT chat-only summaries

---

## 10. Validation

Frontend:
- No errors
- Build success

Backend:
- Type-safe
- Lint pass

Mandatory:
- Claude review

---

## 11. Strict Execution Flow

1. gitnexus analyze --force  
2. Root file reading  
3. GitNexus exploration  
4. Subagent exploration (NEW)  
5. Architecture discovery  
6. Documentation reading  
7. API discovery  
8. Data flow mapping  
9. Create `.full-project/`  
10. Snapshot reuse check  
11. Targeted code reading  
12. Task analysis  
13. Create implementation plan  
14. WAIT approval  
15. Implement  
16. Update docs  
17. Validate  
18. Claude review  
19. Re-analyze  
20. Final verification  
21. Cleanup  
22. Final analyze  

---

## 12. Failure Handling

- STOP immediately  
- Explain clearly  
- DO NOT guess  
- DO NOT partially implement  
