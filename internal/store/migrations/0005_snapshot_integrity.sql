ALTER TABLE idempotency_keys
ADD COLUMN snapshot_digest BLOB
-- Migrate backfills existing v4 rows in the same transaction.
CHECK (snapshot_digest IS NULL OR length(snapshot_digest) = 32);
