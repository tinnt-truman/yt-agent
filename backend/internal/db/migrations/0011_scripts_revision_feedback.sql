-- Stores a pending human-written revision note (e.g. "cảnh 3 dài quá, rút
-- gọn") between the moment a script is sent back to "pending" for revision
-- and the next Step call, which reads it, asks the AI to revise the
-- existing script accordingly, then clears it. NULL/empty means a normal
-- (from-scratch) generation, not a revision.
ALTER TABLE scripts ADD COLUMN IF NOT EXISTS revision_feedback TEXT;
