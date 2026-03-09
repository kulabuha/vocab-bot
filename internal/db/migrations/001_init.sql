PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS collocations (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  phrase        TEXT NOT NULL,
  source_word   TEXT NOT NULL,
  status        TEXT NOT NULL CHECK(status IN ('NEW','LEARNING','MASTERED')),
  level         INTEGER NOT NULL CHECK(level BETWEEN 1 AND 3),
  next_due      INTEGER NOT NULL,
  wrong_streak  INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_collocations_phrase ON collocations(phrase);

CREATE TABLE IF NOT EXISTS exercises (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  chat_id         INTEGER NOT NULL,
  collocation_id  INTEGER NOT NULL,
  level           INTEGER NOT NULL CHECK(level BETWEEN 1 AND 3),
  kind            TEXT NOT NULL CHECK(kind IN ('FILL','MEANING','PARAPHRASE','REFRESH')),
  prompt          TEXT NOT NULL,
  answer_key      TEXT,
  created_at      INTEGER NOT NULL,
  FOREIGN KEY(collocation_id) REFERENCES collocations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_exercises_chat_created ON exercises(chat_id, created_at DESC);

CREATE TABLE IF NOT EXISTS attempts (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  chat_id        INTEGER NOT NULL,
  exercise_id    INTEGER NOT NULL,
  collocation_id INTEGER NOT NULL,
  attempt_level  INTEGER NOT NULL CHECK(attempt_level BETWEEN 1 AND 3),
  kind           TEXT NOT NULL,
  answer         TEXT NOT NULL,
  is_correct     INTEGER NOT NULL CHECK(is_correct IN (0,1)),
  score          INTEGER NOT NULL DEFAULT 0,
  feedback       TEXT,
  created_at     INTEGER NOT NULL,
  FOREIGN KEY(exercise_id) REFERENCES exercises(id) ON DELETE CASCADE,
  FOREIGN KEY(collocation_id) REFERENCES collocations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_attempts_chat_created ON attempts(chat_id, created_at DESC);

CREATE TABLE IF NOT EXISTS chat_state (
  chat_id         INTEGER PRIMARY KEY,
  mode            TEXT NOT NULL CHECK(mode IN ('IDLE','ADDING','TRAINING')),
  refresh_counter INTEGER NOT NULL DEFAULT 0,
  updated_at      INTEGER NOT NULL
);
