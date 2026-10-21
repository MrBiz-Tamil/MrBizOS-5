-- 002_seed_units_and_taxes.sql
-- Seed Centralized Global Units Master

-- 1. WEIGHT (Base unit: KG)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('KG', 'Kilogram', 'kg', 'WEIGHT', 'KG', 1.000000, 3, TRUE),
('G', 'Gram', 'g', 'WEIGHT', 'KG', 0.001000, 3, TRUE),
('MG', 'Milligram', 'mg', 'WEIGHT', 'KG', 0.000001, 3, TRUE),
('QUINTAL', 'Quintal', 'q', 'WEIGHT', 'KG', 100.000000, 2, TRUE),
('TON', 'Metric Ton', 't', 'WEIGHT', 'KG', 1000.000000, 3, TRUE)
ON CONFLICT DO NOTHING;

-- 2. VOLUME (Base unit: LITER)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('L', 'Liter', 'l', 'VOLUME', 'L', 1.000000, 3, TRUE),
('ML', 'Milliliter', 'ml', 'VOLUME', 'L', 0.001000, 3, TRUE),
('KL', 'Kiloliter', 'kl', 'VOLUME', 'L', 1000.000000, 3, TRUE)
ON CONFLICT DO NOTHING;

-- 3. LENGTH (Base unit: METER)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('M', 'Meter', 'm', 'LENGTH', 'M', 1.000000, 3, TRUE),
('MM', 'Millimeter', 'mm', 'LENGTH', 'M', 0.001000, 3, TRUE),
('CM', 'Centimeter', 'cm', 'LENGTH', 'M', 0.010000, 2, TRUE),
('KM', 'Kilometer', 'km', 'LENGTH', 'M', 1000.000000, 3, TRUE)
ON CONFLICT DO NOTHING;

-- 4. AREA (Base unit: SQM)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('SQM', 'Square Meter', 'sq m', 'AREA', 'SQM', 1.000000, 2, TRUE),
('SQMM', 'Square Millimeter', 'sq mm', 'AREA', 'SQM', 0.000001, 2, TRUE),
('SQCM', 'Square Centimeter', 'sq cm', 'AREA', 'SQM', 0.000100, 2, TRUE),
('ACRE', 'Acre', 'ac', 'AREA', 'SQM', 4046.856422, 4, TRUE),
('HECTARE', 'Hectare', 'ha', 'AREA', 'SQM', 10000.000000, 4, TRUE)
ON CONFLICT DO NOTHING;

-- 5. QUANTITY (Base unit: PCS)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('PCS', 'Piece', 'pcs', 'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('UNIT', 'Unit', 'unt', 'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('DOZEN', 'Dozen', 'dz', 'QUANTITY', 'PCS', 12.000000, 0, TRUE),
('PAIR', 'Pair', 'pr', 'QUANTITY', 'PCS', 2.000000, 0, TRUE),
('BOX', 'Box', 'box', 'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('PACK', 'Pack', 'pk', 'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('BOTTLE', 'Bottle', 'btl', 'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('BAG', 'Bag', 'bag', 'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('CARTON', 'Carton', 'ctn', 'QUANTITY', 'PCS', 1.000000, 0, TRUE)
ON CONFLICT DO NOTHING;
