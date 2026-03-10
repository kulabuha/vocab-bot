# Exercise levels spec — collocation training

## Design principles (research-based)

- **Receptive before productive**: Establish form–meaning link first (Level 1: meaning), then controlled production (Level 2: gap-fill), then free production (Level 3–4). Aligns with input-to-output progression in SLA and vocabulary research (e.g. Nation, Schmitt).
- **One task type per level**: Every collocation goes through the same four levels. Same level ⇒ same exercise type for all collocations (e.g. all Level 1 = “explain meaning”).
- **Spaced repetition**: After a correct answer, next review is delayed (e.g. Level 1: 2h, Level 2: 1d, Level 3: 3d, Level 4: 1w). After a wrong answer, item is rescheduled sooner. Intervals are deterministic in code (no LLM).
- **Order and frequency**: Pick the next exercise by (1) due date (`next_due <= now`), then (2) among due items, random or by `next_due` to avoid always the same collocation. Mastered items are occasionally mixed in (e.g. every N sessions) for retention (REFRESH).

---

## Levels (same for every collocation)

Each collocation has a **level** 1–4. Level determines the **single** exercise type for that collocation until the user answers correctly and the collocation advances to the next level. After Level 4 correct → status **MASTERED**; mastered items can later be shown as REFRESH (gap-fill).

| Level | Task type   | Goal (research)                    | What the user sees (prompt) |
|-------|-------------|------------------------------------|-----------------------------|
| **1** | **MEANING** | Receptive: form–meaning link      | “What does *X* mean? Explain in your own words (in English).” |
| **2** | **GAP**     | Controlled production: insert form | “Complete the sentence using *X* (in English): «… __________ …»” |
| **3** | **FILL**    | Free production: use in sentence   | “Use the collocation *X* in one natural sentence (in English).” |
| **4** | **PARAPHRASE** | Transfer: use in new context   | “Rewrite the following using *X* (in English): «…»” |

After **Level 4** correct → collocation becomes **MASTERED**. No Level 5; mastered items are reviewed with **REFRESH** (gap-fill) on a separate schedule.

---

## Task types and prompts (for LLM and bot)

### Level 1 — MEANING (receptive)
- **Prompt (user):**  
  `Level 1 — Explain meaning`  
  `What does *<phrase>* mean? Explain in your own words (in English).`
- **Grading:** Accept any answer that conveys the meaning; no need for the exact phrase. LLM returns `correct_variant` / `native_variant` as full explanation sentences.

### Level 2 — GAP (controlled production)
- **Prompt (user):**  
  When a phrase-specific gap sentence is stored (from the generator’s example):  
  `Level 2 — Fill the gap`  
  `Complete the sentence. The missing part is a collocation that includes the word *<source_word>* (in English). Reply with the full sentence.`  
  `«<sentence with one gap, e.g. "We need to __________ by Friday.">»`  
  The full collocation is **not** shown; the **source word** (e.g. *deadline*) is given as a clue so the user knows which word family to use but must recall the exact phrase (e.g. *meet a deadline*).  
  Fallback (no gap sentence): same as before, with the collocation shown.
- **Grading:** Answer must contain the required collocation (or the missing part); allow small grammatical variations.

### Level 3 — FILL (free production)
- **Prompt (user):**  
  `Level 3 — Use in a sentence`  
  `Use the collocation *<phrase>* in one natural sentence (in English).`
- **Grading:** Collocation must appear in the answer; sentence must be grammatical and natural.

### Level 4 — PARAPHRASE (transfer)
- **Prompt (user):**  
  `Level 4 — Paraphrase`  
  `Rewrite the following using *<phrase>* (in English):`  
  `«<short context sentence>»`
- **Grading:** Collocation must appear; meaning of the given sentence must be preserved.

### REFRESH (mastered only)
- **Prompt (user):**  
  Same as Level 2: when a gap sentence exists, show only the sentence with the blank (collocation not shown). Otherwise show collocation + generic sentence.
- Used only for items already MASTERED, to reinforce retention.

---

## Selection and progression (bot logic)

1. **Next exercise**
   - Prefer **due** items: `status IN ('NEW','LEARNING')` and `next_due <= now`, ordered by `next_due ASC` (and optionally `wrong_streak DESC` so struggling items appear sooner).
   - If none due, take **any learning** items (e.g. `ORDER BY next_due ASC`), limit pool (e.g. 50).
   - From the pool, pick **one at random** (or first by `next_due`) so the same collocation is not always first.
   - Every N sessions (e.g. every 5th), optionally mix in **mastered** items: pick a few at random for REFRESH, then fill the rest with due/learning as above.

2. **Task for chosen collocation**
   - Read collocation’s **level** (1–4) and **status**.
   - If `status == MASTERED` and this slot is for refresh → send **REFRESH** (gap-fill) task.
   - Else send the task for current **level**: 1 → MEANING, 2 → GAP, 3 → FILL, 4 → PARAPHRASE.
   - Store the exercise (chat_id, collocation_id, level, kind, prompt, answer_key).

3. **After user answer**
   - **Correct:**  
     - If level < 4: set level = level + 1, status = LEARNING, `next_due` = now + interval(level).  
     - If level == 4: set status = MASTERED, level = 4, `next_due` = now + long interval (e.g. 7 days).  
     - Set `wrong_streak = 0`.
   - **Wrong:**  
     - Keep level and status; increment `wrong_streak`; set `next_due` = now + short interval (e.g. 10–30 min depending on streak).

---

## Spaced repetition (intervals)

- **After correct:**  
  - Level 1: e.g. 2 hours  
  - Level 2: e.g. 1 day  
  - Level 3: e.g. 3 days  
  - Level 4: e.g. 7 days (then mark MASTERED)  
  - REFRESH (mastered): e.g. 7 days or more  
- **After wrong:**  
  - 1st–2nd wrong: e.g. 10 min  
  - 3rd wrong: e.g. 30 min  
  - 4+ wrong: e.g. 2 hours  

Exact values are configurable in code (e.g. `internal/srs/scheduler.go`).

---

## Summary: 4 levels, 4 task types, same for all

- **Level 1** → MEANING (explain meaning)  
- **Level 2** → GAP (fill the gap)  
- **Level 3** → FILL (use in a sentence)  
- **Level 4** → PARAPHRASE (rewrite with collocation)  
- **MASTERED** → REFRESH (gap-fill for retention)

Per collocation: one task per level in order 1 → 2 → 3 → 4; on correct, that collocation moves to the next level (or to MASTERED after 4). Order of *which* collocation is shown is driven by `next_due` and pool randomization, not by level.
