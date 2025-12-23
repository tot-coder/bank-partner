CREATE TABLE merchant_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_pattern VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    category_code VARCHAR(50) NOT NULL REFERENCES transaction_categories(code) ON DELETE CASCADE,
    mcc_code VARCHAR(10),
    match_type VARCHAR(20) NOT NULL DEFAULT 'EXACT',
    confidence_score DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    usage_count INT NOT NULL DEFAULT 0,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_merchant_mappings_match_type CHECK (match_type IN ('EXACT', 'PARTIAL', 'FUZZY', 'REGEX')),
    CONSTRAINT chk_merchant_mappings_confidence CHECK (confidence_score BETWEEN 0.00 AND 1.00)
);

CREATE INDEX idx_merchant_mappings_pattern ON merchant_mappings(merchant_pattern);
CREATE INDEX idx_merchant_mappings_normalized_name ON merchant_mappings(normalized_name);
CREATE INDEX idx_merchant_mappings_category ON merchant_mappings(category_code);
CREATE INDEX idx_merchant_mappings_mcc ON merchant_mappings(mcc_code);
CREATE INDEX idx_merchant_mappings_is_active ON merchant_mappings(is_active);

-- Seed common merchant patterns for Groceries
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('Walmart', 'Walmart', 'GROCERIES', '5411', 'PARTIAL', 0.95),
('Kroger', 'Kroger', 'GROCERIES', '5411', 'PARTIAL', 0.95),
('Safeway', 'Safeway', 'GROCERIES', '5411', 'PARTIAL', 0.95),
('Whole Foods', 'Whole Foods Market', 'GROCERIES', '5411', 'PARTIAL', 0.95),
('Trader Joe', 'Trader Joes', 'GROCERIES', '5411', 'PARTIAL', 0.95),
('Costco', 'Costco', 'GROCERIES', '5411', 'PARTIAL', 0.95),
('Target', 'Target', 'SHOPPING', '5411', 'PARTIAL', 0.90),
('Aldi', 'Aldi', 'GROCERIES', '5411', 'PARTIAL', 0.95);

-- Seed common merchant patterns for Dining
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('Starbucks', 'Starbucks', 'DINING', '5814', 'PARTIAL', 0.95),
('McDonald', 'McDonalds', 'DINING', '5814', 'PARTIAL', 0.95),
('Chipotle', 'Chipotle', 'DINING', '5812', 'PARTIAL', 0.95),
('Subway', 'Subway', 'DINING', '5814', 'PARTIAL', 0.95),
('Taco Bell', 'Taco Bell', 'DINING', '5814', 'PARTIAL', 0.95),
('Panera', 'Panera Bread', 'DINING', '5814', 'PARTIAL', 0.95),
('Dunkin', 'Dunkin Donuts', 'DINING', '5814', 'PARTIAL', 0.95),
('Pizza Hut', 'Pizza Hut', 'DINING', '5812', 'PARTIAL', 0.95);

-- Seed common merchant patterns for Transportation
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('Uber', 'Uber', 'TRANSPORTATION', '4121', 'PARTIAL', 0.95),
('Lyft', 'Lyft', 'TRANSPORTATION', '4121', 'PARTIAL', 0.95),
('Shell', 'Shell', 'TRANSPORTATION', '5542', 'PARTIAL', 0.95),
('Chevron', 'Chevron', 'TRANSPORTATION', '5542', 'PARTIAL', 0.95),
('Exxon', 'ExxonMobil', 'TRANSPORTATION', '5542', 'PARTIAL', 0.95),
('BP', 'BP', 'TRANSPORTATION', '5542', 'PARTIAL', 0.95),
('Mobil', 'Mobil', 'TRANSPORTATION', '5542', 'PARTIAL', 0.95);

-- Seed common merchant patterns for Entertainment
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('Netflix', 'Netflix', 'ENTERTAINMENT', '7832', 'PARTIAL', 0.95),
('Spotify', 'Spotify', 'ENTERTAINMENT', '5735', 'PARTIAL', 0.95),
('AMC', 'AMC Theaters', 'ENTERTAINMENT', '7832', 'PARTIAL', 0.95),
('Hulu', 'Hulu', 'ENTERTAINMENT', '7832', 'PARTIAL', 0.95),
('Disney', 'Disney Plus', 'ENTERTAINMENT', '7832', 'PARTIAL', 0.90),
('HBO', 'HBO', 'ENTERTAINMENT', '7832', 'PARTIAL', 0.95);

-- Seed common merchant patterns for Shopping
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('Amazon', 'Amazon', 'SHOPPING', '5999', 'PARTIAL', 0.95),
('Best Buy', 'Best Buy', 'SHOPPING', '5732', 'PARTIAL', 0.95),
('Apple', 'Apple Store', 'SHOPPING', '5732', 'PARTIAL', 0.90),
('Home Depot', 'Home Depot', 'SHOPPING', '5211', 'PARTIAL', 0.95),
('Lowes', 'Lowes', 'SHOPPING', '5211', 'PARTIAL', 0.95),
('Ikea', 'IKEA', 'SHOPPING', '5712', 'PARTIAL', 0.95);

-- Seed common merchant patterns for Bills & Utilities
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('AT&T', 'AT&T', 'BILLS_UTILITIES', '4814', 'PARTIAL', 0.95),
('Verizon', 'Verizon', 'BILLS_UTILITIES', '4814', 'PARTIAL', 0.95),
('T-Mobile', 'T-Mobile', 'BILLS_UTILITIES', '4814', 'PARTIAL', 0.95),
('Comcast', 'Comcast', 'BILLS_UTILITIES', '4899', 'PARTIAL', 0.95),
('PG&E', 'Pacific Gas & Electric', 'BILLS_UTILITIES', '4900', 'PARTIAL', 0.95),
('Edison', 'Southern California Edison', 'BILLS_UTILITIES', '4900', 'PARTIAL', 0.90);

-- Seed common merchant patterns for Healthcare
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('CVS', 'CVS Pharmacy', 'HEALTHCARE', '5912', 'PARTIAL', 0.95),
('Walgreens', 'Walgreens', 'HEALTHCARE', '5912', 'PARTIAL', 0.95),
('Rite Aid', 'Rite Aid', 'HEALTHCARE', '5912', 'PARTIAL', 0.95);

-- Seed common merchant patterns for Travel
INSERT INTO merchant_mappings (merchant_pattern, normalized_name, category_code, mcc_code, match_type, confidence_score) VALUES
('Delta', 'Delta Air Lines', 'TRAVEL', '3000', 'PARTIAL', 0.95),
('United', 'United Airlines', 'TRAVEL', '3000', 'PARTIAL', 0.95),
('American Airlines', 'American Airlines', 'TRAVEL', '3000', 'PARTIAL', 0.95),
('Southwest', 'Southwest Airlines', 'TRAVEL', '3000', 'PARTIAL', 0.95),
('Marriott', 'Marriott', 'TRAVEL', '7011', 'PARTIAL', 0.95),
('Hilton', 'Hilton', 'TRAVEL', '7011', 'PARTIAL', 0.95),
('Hyatt', 'Hyatt', 'TRAVEL', '7011', 'PARTIAL', 0.95);
