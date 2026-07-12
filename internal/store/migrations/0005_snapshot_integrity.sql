ALTER TABLE idempotency_keys
ADD COLUMN snapshot_digest BLOB
-- Existing v4 rows remain readable as corrupt records and fail replay safely.
CHECK (snapshot_digest IS NULL OR length(snapshot_digest) = 32);
