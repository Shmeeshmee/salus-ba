-- +goose Up
-- Wave 6 integration (docs/clinic-cms/WAVE6.md U2, the owner's decision of 2026-11-25): a receipt of an in-person visit
-- names where the service was given, the visit's location with its address, copied at issue like the seller. The seller
-- stays the clinic's registered details (its address is a fiscal matter). No personal data: a clinic's site. Null for
-- a receipt of a video visit, of none, and for every receipt issued before. invoice_guard_change already refuses a
-- change of any column but the fiscal ones, this one included.
ALTER TABLE invoice ADD COLUMN place_of_service text CHECK (place_of_service <> '');
