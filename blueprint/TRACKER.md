# Pulp Development Tracker

## Rules

1. **Read the phase file** before starting work
2. **Build end-to-end** - no skeletons, no placeholders, working code
3. **Test before marking done** - it must run
4. **Commit + push** after completing each phase
5. **Mark DONE here** when phase is complete
6. **Never read finished phases** - move forward only
7. **Never guess always check if you actually finished a phase** - same for everything

---

## Phases

### Phase 1: Foundation
**File:** `blueprint/phases/01-foundation.md`
**Status:** DONE
**Goal:** Go module + basic TUI that launches and quits
**Deliverable:** `pulp` binary that shows welcome screen, responds to Esc

---

### Phase 2: Configuration
**File:** `blueprint/phases/02-configuration.md`
**Status:** PENDING
**Goal:** Config system + first-run setup wizard
**Deliverable:** Setup wizard works, saves config to `~/.config/pulp/config.yaml`

---

### Phase 3: Provider System
**File:** `blueprint/phases/03-providers.md`
**Status:** PENDING
**Goal:** Provider interface + Ollama implementation
**Deliverable:** Can connect to Ollama, send message, get response

---

### Phase 4: Document Loading
**File:** `blueprint/phases/04-documents.md`
**Status:** PENDING
**Goal:** Python bridge + document loading UI
**Deliverable:** Load PDF/DOCX, show preview in TUI

---

### Phase 5: Intent Parser
**File:** `blueprint/phases/05-intent.md`
**Status:** PENDING
**Goal:** Natural language instruction parsing
**Deliverable:** "summarize for my boss" -> structured Intent

---

### Phase 6: Pipeline
**File:** `blueprint/phases/06-pipeline.md`
**Status:** PENDING
**Goal:** Processing pipeline with animated progress
**Deliverable:** Full flow from document to extraction with progress UI

---

### Phase 7: Writer + Output
**File:** `blueprint/phases/07-writer.md`
**Status:** PENDING
**Goal:** Writer component with streaming output
**Deliverable:** See tokens stream into result view

---

### Phase 8: Session
**File:** `blueprint/phases/08-session.md`
**Status:** PENDING
**Goal:** Conversation history + follow-ups
**Deliverable:** "make it shorter" works after initial result

---

### Phase 9: All Providers
**File:** `blueprint/phases/09-all-providers.md`
**Status:** PENDING
**Goal:** Groq, OpenAI, Anthropic, OpenRouter, Custom
**Deliverable:** All providers work, can switch in settings

---

### Phase 10: Release
**File:** `blueprint/phases/10-release.md`
**Status:** PENDING
**Goal:** Polish + packaging
**Deliverable:** `brew install pulp` works, README complete

---

## Progress Log

<!-- Add entries when completing phases -->
<!-- Format: YYYY-MM-DD | Phase X | Commit SHA | Notes -->
2026-01-29 | Phase 1 | TBD | Foundation complete - TUI with centered logo, status bar, Esc to quit
