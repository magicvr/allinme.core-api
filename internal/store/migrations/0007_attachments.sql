CREATE TABLE attachments (
    id TEXT PRIMARY KEY CHECK (
        typeof(id) = 'text' AND
        length(id) = 36 AND
        substr(id, 1, 4) = 'att_' AND
        substr(id, 5) NOT GLOB '*[^0-9a-f]*'
    ),
    created_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (typeof(status) = 'text' AND status IN ('UPLOADED', 'BOUND', 'DELETING')),
    file_name TEXT NOT NULL CHECK (typeof(file_name) = 'text' AND length(CAST(file_name AS BLOB)) BETWEEN 1 AND 255),
    storage_key TEXT NOT NULL UNIQUE CHECK (
        typeof(storage_key) = 'text' AND
        length(storage_key) = 36 AND
        substr(storage_key, 1, 4) = 'att_' AND
        substr(storage_key, 5) NOT GLOB '*[^0-9a-f]*'
    ),
    content_type TEXT NOT NULL CHECK (typeof(content_type) = 'text' AND content_type IN ('application/pdf', 'image/png', 'image/jpeg')),
    size_bytes INTEGER NOT NULL CHECK (typeof(size_bytes) = 'integer' AND size_bytes BETWEEN 1 AND 10485760),
    sha256 BLOB NOT NULL CHECK (typeof(sha256) = 'blob' AND length(sha256) = 32),
    expires_at TEXT CHECK (
        expires_at IS NULL OR (
            typeof(expires_at) = 'text' AND
            length(expires_at) = 20 AND
            expires_at GLOB '????-??-??T??:??:??Z' AND
            strftime('%Y-%m-%dT%H:%M:%SZ', expires_at) IS NOT NULL AND
            strftime('%Y-%m-%dT%H:%M:%SZ', expires_at) = expires_at
        )
    ),
    created_at TEXT NOT NULL CHECK (
        typeof(created_at) = 'text' AND
        length(created_at) = 20 AND
        created_at GLOB '????-??-??T??:??:??Z' AND
        strftime('%Y-%m-%dT%H:%M:%SZ', created_at) IS NOT NULL AND
        strftime('%Y-%m-%dT%H:%M:%SZ', created_at) = created_at
    ),
    updated_at TEXT NOT NULL CHECK (
        typeof(updated_at) = 'text' AND
        length(updated_at) = 20 AND
        updated_at GLOB '????-??-??T??:??:??Z' AND
        strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) IS NOT NULL AND
        strftime('%Y-%m-%dT%H:%M:%SZ', updated_at) = updated_at
    ),
    CHECK (
        (status = 'UPLOADED' AND expires_at IS NOT NULL) OR
        (status = 'BOUND' AND expires_at IS NULL) OR
        status = 'DELETING'
    )
);

CREATE TABLE order_attachments (
    attachment_id TEXT PRIMARY KEY REFERENCES attachments(id) ON DELETE RESTRICT,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (typeof(position) = 'integer' AND position BETWEEN 0 AND 9),
    bound_at TEXT NOT NULL CHECK (
        typeof(bound_at) = 'text' AND
        length(bound_at) = 20 AND
        bound_at GLOB '????-??-??T??:??:??Z' AND
        strftime('%Y-%m-%dT%H:%M:%SZ', bound_at) IS NOT NULL AND
        strftime('%Y-%m-%dT%H:%M:%SZ', bound_at) = bound_at
    ),
    UNIQUE(order_id, position)
);

CREATE INDEX attachments_created_by_status_idx ON attachments(created_by, status);
CREATE INDEX attachments_status_expires_at_idx ON attachments(status, expires_at);
CREATE INDEX order_attachments_order_position_idx ON order_attachments(order_id, position);
