CREATE TABLE refunds (
    id TEXT PRIMARY KEY CHECK (
        length(id) = 36 AND
        substr(id, 1, 4) = 'rfd_' AND
        substr(id, 5) NOT GLOB '*[^0-9a-f]*'
    ),
    order_id TEXT NOT NULL REFERENCES orders(id),
    amount INTEGER NOT NULL CHECK (amount BETWEEN 1 AND 9999999999),
    reason TEXT NOT NULL CHECK (length(CAST(reason AS BLOB)) BETWEEN 1 AND 500),
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'REJECTED', 'COMPLETED')),
    version INTEGER NOT NULL CHECK (version >= 1),
    requested_by TEXT NOT NULL REFERENCES users(id),
    decided_by TEXT REFERENCES users(id),
    created_at TEXT NOT NULL CHECK (length(created_at) = 20 AND substr(created_at, 20, 1) = 'Z'),
    updated_at TEXT NOT NULL CHECK (length(updated_at) = 20 AND substr(updated_at, 20, 1) = 'Z'),
    decided_at TEXT CHECK (decided_at IS NULL OR (length(decided_at) = 20 AND substr(decided_at, 20, 1) = 'Z')),
    CHECK (
        (status = 'PENDING' AND version = 1 AND decided_by IS NULL AND decided_at IS NULL AND updated_at = created_at) OR
        (status IN ('REJECTED', 'COMPLETED') AND version = 2 AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND updated_at = decided_at)
    )
);

CREATE TABLE refund_idempotency_keys (
    principal_user_id TEXT NOT NULL REFERENCES users(id),
    method TEXT NOT NULL CHECK (method = 'POST'),
    operation TEXT NOT NULL CHECK (length(CAST(operation AS BLOB)) BETWEEN 1 AND 200),
    order_id TEXT NOT NULL REFERENCES orders(id),
    idempotency_key TEXT NOT NULL CHECK (length(CAST(idempotency_key AS BLOB)) BETWEEN 1 AND 128),
    request_digest BLOB NOT NULL CHECK (length(request_digest) = 32),
    refund_id TEXT NOT NULL REFERENCES refunds(id),
    snapshot_version INTEGER NOT NULL CHECK (snapshot_version >= 1),
    snapshot_json TEXT NOT NULL CHECK (json_valid(snapshot_json)),
    snapshot_digest BLOB NOT NULL CHECK (length(snapshot_digest) = 32),
    created_at TEXT NOT NULL CHECK (length(created_at) = 20 AND substr(created_at, 20, 1) = 'Z'),
    PRIMARY KEY (principal_user_id, method, operation, order_id, idempotency_key)
);

CREATE INDEX refunds_order_status_idx ON refunds(order_id, status);
CREATE INDEX refunds_status_created_at_id_idx ON refunds(status, created_at, id);
CREATE INDEX refunds_order_created_at_id_idx ON refunds(order_id, created_at, id);
CREATE INDEX refunds_status_decided_at_idx ON refunds(status, decided_at);
CREATE INDEX refund_idempotency_keys_refund_id_idx ON refund_idempotency_keys(refund_id);
