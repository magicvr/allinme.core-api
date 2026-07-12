CREATE TABLE orders (
    id TEXT PRIMARY KEY CHECK (
        length(id) = 36 AND
        substr(id, 1, 4) = 'ord_' AND
        substr(id, 5) NOT GLOB '*[^0-9a-f]*'
    ),
    customer_name TEXT NOT NULL CHECK (length(CAST(customer_name AS BLOB)) BETWEEN 1 AND 120),
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'CONFIRMED', 'FULFILLING', 'SHIPPED', 'COMPLETED', 'CANCELLED')),
    payment_status TEXT NOT NULL CHECK (payment_status IN ('UNPAID', 'PAID', 'PARTIALLY_REFUNDED', 'REFUNDED')),
    currency TEXT NOT NULL CHECK (currency = 'CNY'),
    total_amount INTEGER NOT NULL CHECK (total_amount BETWEEN 1 AND 9999999999),
    version INTEGER NOT NULL CHECK (version >= 1),
    created_at TEXT NOT NULL CHECK (length(created_at) = 20 AND substr(created_at, 20, 1) = 'Z'),
    updated_at TEXT NOT NULL CHECK (length(updated_at) = 20 AND substr(updated_at, 20, 1) = 'Z')
);

CREATE TABLE order_items (
    id TEXT PRIMARY KEY CHECK (
        length(id) = 36 AND
        substr(id, 1, 4) = 'itm_' AND
        substr(id, 5) NOT GLOB '*[^0-9a-f]*'
    ),
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    sku TEXT NOT NULL CHECK (length(CAST(sku AS BLOB)) BETWEEN 1 AND 64),
    name TEXT NOT NULL CHECK (length(CAST(name AS BLOB)) BETWEEN 1 AND 160),
    quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 10000),
    unit_price INTEGER NOT NULL CHECK (unit_price BETWEEN 1 AND 9999999999),
    UNIQUE(order_id, position)
);

CREATE INDEX orders_created_at_id_idx ON orders(created_at, id);
CREATE INDEX orders_updated_at_id_idx ON orders(updated_at, id);
CREATE INDEX orders_total_amount_id_idx ON orders(total_amount, id);
CREATE INDEX orders_customer_name_id_idx ON orders(customer_name, id);
CREATE INDEX orders_status_id_idx ON orders(status, id);
CREATE INDEX orders_payment_status_idx ON orders(payment_status);
CREATE INDEX order_items_order_position_idx ON order_items(order_id, position);
CREATE INDEX order_items_sku_idx ON order_items(sku);
