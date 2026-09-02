-- +goose Up
ALTER TABLE cart_items
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX cart_items_user_id_item_id_unique ON cart_items (user_id, item_id);

-- +goose Down
DROP INDEX IF EXISTS cart_items_user_id_item_id_unique;

ALTER TABLE cart_items
    DROP COLUMN user_id;
