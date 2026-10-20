-- +goose Up
-- Extensions the schema relies on: btree_gist for exclusion constraints on time ranges (slot overlaps),
-- pgcrypto for cryptographic functions.
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
