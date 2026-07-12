CREATE TABLE idempotency_keys (
    principal_user_id TEXT NOT NULL,
    method TEXT NOT NULL,
    route TEXT NOT NULL,
    idempotency_key TEXT NOT NULL CHECK (length(CAST(idempotency_key AS BLOB)) BETWEEN 1 AND 128),
    request_digest BLOB NOT NULL CHECK (length(request_digest) = 32),
    order_id TEXT NOT NULL CHECK (
        length(order_id) = 36 AND
        substr(order_id, 1, 4) = 'ord_' AND
        substr(order_id, 5) NOT GLOB '*[^0-9a-f]*'
    ),
    snapshot_version INTEGER NOT NULL CHECK (snapshot_version >= 1),
    snapshot_json TEXT NOT NULL CHECK (json_valid(snapshot_json)),
    created_at TEXT NOT NULL CHECK (length(created_at) = 20 AND substr(created_at, 20, 1) = 'Z'),
    PRIMARY KEY (principal_user_id, method, route, idempotency_key)
);

CREATE INDEX idempotency_keys_order_id_idx ON idempotency_keys(order_id);
CREATE INDEX idempotency_keys_created_at_idx ON idempotency_keys(created_at);
