-- Per-user collocations: each user only sees their own. Backfill chat_id from exercises.
PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS collocations_new (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  chat_id       INTEGER NOT NULL,
  phrase        TEXT NOT NULL,
  source_word   TEXT NOT NULL,
  status        TEXT NOT NULL CHECK(status IN ('NEW','LEARNING','MASTERED')),
  level         INTEGER NOT NULL CHECK(level BETWEEN 1 AND 4),
  next_due      INTEGER NOT NULL,
  wrong_streak  INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
INSERT INTO collocations_new
SELECT c.id, COALESCE((SELECT e.chat_id FROM exercises e WHERE e.collocation_id = c.id LIMIT 1), 0),
       c.phrase, c.source_word, c.status, c.level, c.next_due, c.wrong_streak, c.created_at, c.updated_at
FROM collocations c;
DROP TABLE collocations;
ALTER TABLE collocations_new RENAME TO collocations;
CREATE UNIQUE INDEX IF NOT EXISTS idx_collocations_chat_phrase ON collocations(chat_id, phrase);

PRAGMA foreign_keys = ON;
