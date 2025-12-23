CREATE TABLE transaction_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_category_code VARCHAR(50) NULL REFERENCES transaction_categories(code) ON DELETE SET NULL,
    icon VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transaction_categories_code ON transaction_categories(code);
CREATE INDEX idx_transaction_categories_is_active ON transaction_categories(is_active);
CREATE INDEX idx_transaction_categories_parent_category ON transaction_categories(parent_category_code);

-- Seed standard categories
INSERT INTO transaction_categories (code, name, description, display_order) VALUES
('GROCERIES', 'Groceries', 'Supermarkets, grocery stores, and food shopping', 1),
('DINING', 'Dining & Restaurants', 'Restaurants, cafes, fast food, and food delivery', 2),
('TRANSPORTATION', 'Transportation', 'Gas stations, public transit, ride-sharing, parking', 3),
('ENTERTAINMENT', 'Entertainment', 'Movies, streaming services, concerts, events', 4),
('SHOPPING', 'Shopping', 'Retail stores, online shopping, clothing, electronics', 5),
('BILLS_UTILITIES', 'Bills & Utilities', 'Electric, gas, water, internet, phone bills', 6),
('HEALTHCARE', 'Healthcare', 'Medical services, pharmacies, health insurance', 7),
('EDUCATION', 'Education', 'Schools, universities, online courses, books', 8),
('TRAVEL', 'Travel', 'Airlines, hotels, car rentals, travel booking', 9),
('ATM_CASH', 'ATM & Cash', 'ATM withdrawals, cash deposits, cash advances', 10),
('INCOME', 'Income', 'Salary, wages, direct deposits, refunds', 11),
('FEES', 'Fees & Charges', 'Bank fees, service charges, overdraft fees', 12),
('OTHER', 'Other', 'Uncategorized or miscellaneous transactions', 99);
