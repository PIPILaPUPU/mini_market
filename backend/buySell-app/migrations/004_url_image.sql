-- +goose Up
ALTER TABLE items
    ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE items
    DROP COLUMN image_url;
