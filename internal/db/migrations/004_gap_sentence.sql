-- Add gap_sentence for Level 2 GAP (and REFRESH): phrase-specific sentence with blank, so we don't show the collocation.
ALTER TABLE collocations ADD COLUMN gap_sentence TEXT DEFAULT '';
