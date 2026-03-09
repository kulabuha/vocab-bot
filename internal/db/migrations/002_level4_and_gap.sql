-- Extend levels to 1..4 and add GAP exercise kind (see docs/EXERCISE_LEVELS_SPEC.md).
PRAGMA foreign_keys = OFF;

-- Collocations: allow level 4
CREATE TABLE IF NOT EXISTS collocations_new (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  phrase        TEXT NOT NULL,
  source_word   TEXT NOT NULL,
  status        TEXT NOT NULL CHECK(status IN ('NEW','LEARNING','MASTERED')),
  level         INTEGER NOT NULL CHECK(level BETWEEN 1 AND 4),
  next_due      INTEGER NOT NULL,
  wrong_streak  INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
INSERT INTO collocations_new SELECT id, phrase, source_word, status, level, next_due, wrong_streak, created_at, updated_at FROM collocations;
DROP TABLE collocations;
ALTER TABLE collocations_new RENAME TO collocations;
CREATE UNIQUE INDEX IF NOT EXISTS idx_collocations_phrase ON collocations(phrase);

-- Exercises: allow level 4 and kind GAP
CREATE TABLE IF NOT EXISTS exercises_new (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  chat_id         INTEGER NOT NULL,
  collocation_id  INTEGER NOT NULL,
  level           INTEGER NOT NULL CHECK(level BETWEEN 1 AND 4),
  kind            TEXT NOT NULL CHECK(kind IN ('FILL','MEANING','PARAPHRASE','REFRESH','GAP')),
  prompt          TEXT NOT NULL,
  answer_key      TEXT,
  created_at      INTEGER NOT NULL,
  FOREIGN KEY(collocation_id) REFERENCES collocations(id) ON DELETE CASCADE
);
INSERT INTO exercises_new SELECT id, chat_id, collocation_id, level, kind, prompt, answer_key, created_at FROM exercises;
DROP TABLE exercises;
ALTER TABLE exercises_new RENAME TO exercises;
CREATE INDEX IF NOT EXISTS idx_exercises_chat_created ON exercises(chat_id, created_at DESC);

-- Attempts: allow attempt_level 1..4
CREATE TABLE IF NOT EXISTS attempts_new (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  chat_id        INTEGER NOT NULL,
  exercise_id    INTEGER NOT NULL,
  collocation_id INTEGER NOT NULL,
  attempt_level  INTEGER NOT NULL CHECK(attempt_level BETWEEN 1 AND 4),
  kind           TEXT NOT NULL,
  answer         TEXT NOT NULL,
  is_correct     INTEGER NOT NULL CHECK(is_correct IN (0,1)),
  score          INTEGER NOT NULL DEFAULT 0,
  feedback       TEXT,
  created_at     INTEGER NOT NULL,
  FOREIGN KEY(exercise_id) REFERENCES exercises(id) ON DELETE CASCADE,
  FOREIGN KEY(collocation_id) REFERENCES collocations(id) ON DELETE CASCADE
);
INSERT INTO attempts_new SELECT id, chat_id, exercise_id, collocation_id, attempt_level, kind, answer, is_correct, score, feedback, created_at FROM attempts;
DROP TABLE attempts;
ALTER TABLE attempts_new RENAME TO attempts;
CREATE INDEX IF NOT EXISTS idx_attempts_chat_created ON attempts(chat_id, created_at DESC);

PRAGMA foreign_keys = ON;
