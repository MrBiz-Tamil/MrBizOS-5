-- 017_invoice_item_line_order.sql
-- Invoice lines used to be returned ORDER BY random UUID, so printed bills did not follow the order in
-- which items were entered (important for restaurant KOT-style bills and room-charge folios).
ALTER TABLE invoice_items ADD COLUMN IF NOT EXISTS line_no INT NOT NULL DEFAULT 0;
