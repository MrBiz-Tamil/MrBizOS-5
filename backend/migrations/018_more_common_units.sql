-- 018_more_common_units.sql
-- Additional global units that Indian small businesses use every day. Invoices reject unknown unit codes,
-- and only a platform admin can add units, so shops (hardware, food service, hospitality, retail) were
-- blocked from billing in NOS, ROLL, SET, SQFT, PLATE, NIGHT ... Idempotent; existing codes are untouched.

-- LENGTH (base: M)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('MTR',  'Meter (alias)', 'mtr', 'LENGTH', 'M', 1.000000, 3, TRUE),
('FT',   'Foot',          'ft',  'LENGTH', 'M', 0.304800, 2, TRUE),
('INCH', 'Inch',          'in',  'LENGTH', 'M', 0.025400, 2, TRUE),
('YARD', 'Yard',          'yd',  'LENGTH', 'M', 0.914400, 2, TRUE)
ON CONFLICT DO NOTHING;

-- AREA (base: SQM)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('SQFT', 'Square Foot', 'sq ft', 'AREA', 'SQM', 0.092903, 2, TRUE),
('SQYD', 'Square Yard', 'sq yd', 'AREA', 'SQM', 0.836127, 2, TRUE)
ON CONFLICT DO NOTHING;

-- QUANTITY (base: PCS)
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('PC',     'Piece (alias)', 'pc',     'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('NOS',    'Numbers',       'nos',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('SET',    'Set',           'set',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('ROLL',   'Roll',          'roll',   'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('BUNDLE', 'Bundle',        'bdl',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('COIL',   'Coil',          'coil',   'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('SHEET',  'Sheet',         'sht',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('TIN',    'Tin',           'tin',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('CAN',    'Can',           'can',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('DRUM',   'Drum',          'drum',   'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('PACKET', 'Packet',        'pkt',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('STRIP',  'Strip',         'strip',  'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('TABLET', 'Tablet',        'tab',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('VIAL',   'Vial',          'vial',   'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('PLATE',  'Plate',         'plate',  'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('PORTION','Portion',       'ptn',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('GLASS',  'Glass',         'glass',  'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('CUP',    'Cup',           'cup',    'QUANTITY', 'PCS', 1.000000, 0, TRUE),
('SLICE',  'Slice',         'slice',  'QUANTITY', 'PCS', 1.000000, 0, TRUE)
ON CONFLICT DO NOTHING;

-- TIME (base: HOUR) for lodging / services
INSERT INTO units (code, name, symbol, category, base_unit_code, conversion_factor, decimal_precision, is_system) VALUES
('HOUR',  'Hour',  'hr',    'TIME', 'HOUR', 1.000000,   2, TRUE),
('DAY',   'Day',   'day',   'TIME', 'HOUR', 24.000000,  2, TRUE),
('NIGHT', 'Night', 'night', 'TIME', 'HOUR', 24.000000,  0, TRUE),
('WEEK',  'Week',  'wk',    'TIME', 'HOUR', 168.000000, 2, TRUE),
('MONTH', 'Month', 'mo',    'TIME', 'HOUR', 720.000000, 2, TRUE),
('YEAR',  'Year',  'yr',    'TIME', 'HOUR', 8760.000000,2, TRUE)
ON CONFLICT DO NOTHING;
