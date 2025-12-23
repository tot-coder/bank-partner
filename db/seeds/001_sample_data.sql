-- Seed data for Array Banking API (Interview Edition)
-- This file is idempotent and can be run multiple times

-- Clear existing seed data (if any)
-- Note: This uses ON CONFLICT to make inserts idempotent
-- Passwords are hashed using bcrypt with password: "Password123!"

-- Insert sample users (customers)
INSERT INTO users (id, email, password_hash, first_name, last_name, role, created_at, updated_at)
VALUES
    ('a0000000-0000-0000-0000-000000000001', 'john.doe@example.com', '$2a$10$I4amrz.fROvn2g.yDij7cOUFVnMkFAf2Y.uixaozDJtcJojB220We', 'John', 'Doe', 'admin', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('a0000000-0000-0000-0000-000000000002', 'jane.smith@example.com', '$2a$10$I4amrz.fROvn2g.yDij7cOUFVnMkFAf2Y.uixaozDJtcJojB220We', 'Jane', 'Smith', 'customer', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('a0000000-0000-0000-0000-000000000003', 'bob.johnson@example.com', '$2a$10$I4amrz.fROvn2g.yDij7cOUFVnMkFAf2Y.uixaozDJtcJojB220We', 'Bob', 'Johnson', 'customer', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('a0000000-0000-0000-0000-000000000004', 'alice.williams@example.com', '$2a$10$I4amrz.fROvn2g.yDij7cOUFVnMkFAf2Y.uixaozDJtcJojB220We', 'Alice', 'Williams', 'customer', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('a0000000-0000-0000-0000-000000000005', 'charlie.brown@example.com', '$2a$10$I4amrz.fROvn2g.yDij7cOUFVnMkFAf2Y.uixaozDJtcJojB220We', 'Charlie', 'Brown', 'customer', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (email) DO NOTHING;

-- Insert sample accounts
INSERT INTO accounts (id, account_number, user_id, account_type, balance, status, currency, interest_rate, created_at, updated_at)
VALUES
    -- John Doe's accounts
    ('b0000000-0000-0000-0000-000000000001', '1012345678', 'a0000000-0000-0000-0000-000000000001', 'checking', 5000.00, 'active', 'USD', 0.0000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('b0000000-0000-0000-0000-000000000002', '2012345678', 'a0000000-0000-0000-0000-000000000001', 'savings', 15000.00, 'active', 'USD', 0.0150, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Jane Smith's accounts
    ('b0000000-0000-0000-0000-000000000003', '1023456789', 'a0000000-0000-0000-0000-000000000002', 'checking', 3500.50, 'active', 'USD', 0.0000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('b0000000-0000-0000-0000-000000000004', '2023456789', 'a0000000-0000-0000-0000-000000000002', 'savings', 25000.00, 'active', 'USD', 0.0150, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('b0000000-0000-0000-0000-000000000005', '3023456789', 'a0000000-0000-0000-0000-000000000002', 'money_market', 50000.00, 'active', 'USD', 0.0250, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Bob Johnson's accounts
    ('b0000000-0000-0000-0000-000000000006', '1034567890', 'a0000000-0000-0000-0000-000000000003', 'checking', 1200.75, 'active', 'USD', 0.0000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('b0000000-0000-0000-0000-000000000007', '2034567890', 'a0000000-0000-0000-0000-000000000003', 'savings', 8000.00, 'active', 'USD', 0.0150, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Alice Williams's accounts
    ('b0000000-0000-0000-0000-000000000008', '1045678901', 'a0000000-0000-0000-0000-000000000004', 'checking', 2800.00, 'active', 'USD', 0.0000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('b0000000-0000-0000-0000-000000000009', '3045678901', 'a0000000-0000-0000-0000-000000000004', 'money_market', 35000.00, 'active', 'USD', 0.0250, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

    -- Charlie Brown's account
    ('b0000000-0000-0000-0000-000000000010', '1056789012', 'a0000000-0000-0000-0000-000000000005', 'checking', 750.25, 'active', 'USD', 0.0000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (account_number) DO NOTHING;

-- Insert sample transactions
INSERT INTO transactions (id, account_id, transaction_type, amount, balance_before, balance_after, description, reference, status, category, created_at, updated_at, processed_at)
VALUES
    -- John Doe's checking account transactions
    ('c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'credit', 5000.00, 0.00, 5000.00, 'Initial deposit', 'TXN-INIT-001', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days'),
    ('c0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000001', 'debit', 150.00, 5000.00, 4850.00, 'Grocery shopping at Whole Foods', 'TXN-DEB-001', 'completed', 'GROCERIES', CURRENT_TIMESTAMP - INTERVAL '5 days', CURRENT_TIMESTAMP - INTERVAL '5 days', CURRENT_TIMESTAMP - INTERVAL '5 days'),
    ('c0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000001', 'credit', 200.00, 4850.00, 5050.00, 'Refund from online purchase', 'TXN-REF-001', 'completed', 'OTHER', CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days'),
    ('c0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000001', 'debit', 50.00, 5050.00, 5000.00, 'ATM Withdrawal', 'TXN-ATM-001', 'completed', 'ATM_CASH', CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day'),

    -- John Doe's savings account transactions
    ('c0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000002', 'credit', 15000.00, 0.00, 15000.00, 'Initial savings deposit', 'TXN-INIT-002', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days'),

    -- Jane Smith's checking account transactions
    ('c0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000003', 'credit', 3500.50, 0.00, 3500.50, 'Direct Deposit - Salary', 'TXN-INIT-003', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '15 days', CURRENT_TIMESTAMP - INTERVAL '15 days', CURRENT_TIMESTAMP - INTERVAL '15 days'),
    ('c0000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000003', 'debit', 45.00, 3500.50, 3455.50, 'Coffee at Starbucks', 'TXN-DEB-002', 'completed', 'DINING', CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days'),
    ('c0000000-0000-0000-0000-000000000008', 'b0000000-0000-0000-0000-000000000003', 'debit', 80.00, 3455.50, 3375.50, 'Gas at Shell Station', 'TXN-DEB-003', 'completed', 'TRANSPORTATION', CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day'),

    -- Jane Smith's savings account transactions
    ('c0000000-0000-0000-0000-000000000009', 'b0000000-0000-0000-0000-000000000004', 'credit', 25000.00, 0.00, 25000.00, 'Initial savings deposit', 'TXN-INIT-004', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days'),

    -- Jane Smith's money market account transactions
    ('c0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000005', 'credit', 50000.00, 0.00, 50000.00, 'Initial money market deposit', 'TXN-INIT-005', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days'),

    -- Bob Johnson's checking account transactions
    ('c0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000006', 'credit', 1200.75, 0.00, 1200.75, 'Direct Deposit - Salary', 'TXN-INIT-006', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '7 days'),
    ('c0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000006', 'debit', 120.00, 1200.75, 1080.75, 'Grocery shopping at Walmart', 'TXN-DEB-004', 'completed', 'GROCERIES', CURRENT_TIMESTAMP - INTERVAL '4 days', CURRENT_TIMESTAMP - INTERVAL '4 days', CURRENT_TIMESTAMP - INTERVAL '4 days'),

    -- Bob Johnson's savings account transactions
    ('c0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000007', 'credit', 8000.00, 0.00, 8000.00, 'Initial savings deposit', 'TXN-INIT-007', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days'),

    -- Alice Williams's checking account transactions
    ('c0000000-0000-0000-0000-000000000014', 'b0000000-0000-0000-0000-000000000008', 'credit', 2800.00, 0.00, 2800.00, 'Initial deposit', 'TXN-INIT-008', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '20 days', CURRENT_TIMESTAMP - INTERVAL '20 days', CURRENT_TIMESTAMP - INTERVAL '20 days'),
    ('c0000000-0000-0000-0000-000000000015', 'b0000000-0000-0000-0000-000000000008', 'debit', 15.00, 2800.00, 2785.00, 'Netflix subscription', 'TXN-DEB-005', 'completed', 'ENTERTAINMENT', CURRENT_TIMESTAMP - INTERVAL '10 days', CURRENT_TIMESTAMP - INTERVAL '10 days', CURRENT_TIMESTAMP - INTERVAL '10 days'),

    -- Alice Williams's money market account transactions
    ('c0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000009', 'credit', 35000.00, 0.00, 35000.00, 'Initial money market deposit', 'TXN-INIT-009', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days', CURRENT_TIMESTAMP - INTERVAL '30 days'),

    -- Charlie Brown's checking account transactions
    ('c0000000-0000-0000-0000-000000000017', 'b0000000-0000-0000-0000-000000000010', 'credit', 750.25, 0.00, 750.25, 'Initial deposit', 'TXN-INIT-010', 'completed', 'INCOME', CURRENT_TIMESTAMP - INTERVAL '10 days', CURRENT_TIMESTAMP - INTERVAL '10 days', CURRENT_TIMESTAMP - INTERVAL '10 days'),
    ('c0000000-0000-0000-0000-000000000018', 'b0000000-0000-0000-0000-000000000010', 'debit', 35.00, 750.25, 715.25, 'Mobile payment - Uber', 'TXN-DEB-006', 'completed', 'TRANSPORTATION', CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days', CURRENT_TIMESTAMP - INTERVAL '3 days')
ON CONFLICT (id) DO NOTHING;

-- Insert sample transfers
INSERT INTO transfers (id, from_account_id, to_account_id, amount, description, idempotency_key, status, debit_transaction_id, credit_transaction_id, created_at, updated_at, completed_at)
VALUES
    -- Transfer from John Doe's checking to savings
    ('d0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002', 500.00, 'Transfer to savings', 'IDEMPOT-TRANSFER-001', 'completed', NULL, NULL, CURRENT_TIMESTAMP - INTERVAL '10 days', CURRENT_TIMESTAMP - INTERVAL '10 days', CURRENT_TIMESTAMP - INTERVAL '10 days'),

    -- Transfer from Jane Smith's checking to savings
    ('d0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000004', 1000.00, 'Monthly savings transfer', 'IDEMPOT-TRANSFER-002', 'completed', NULL, NULL, CURRENT_TIMESTAMP - INTERVAL '8 days', CURRENT_TIMESTAMP - INTERVAL '8 days', CURRENT_TIMESTAMP - INTERVAL '8 days'),

    -- Transfer from Jane Smith to Alice Williams
    ('d0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000008', 200.00, 'Payment for dinner', 'IDEMPOT-TRANSFER-003', 'completed', NULL, NULL, CURRENT_TIMESTAMP - INTERVAL '5 days', CURRENT_TIMESTAMP - INTERVAL '5 days', CURRENT_TIMESTAMP - INTERVAL '5 days'),

    -- Pending transfer from Bob Johnson
    ('d0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000001', 100.00, 'Payment to John', 'IDEMPOT-TRANSFER-004', 'pending', NULL, NULL, CURRENT_TIMESTAMP - INTERVAL '1 hour', CURRENT_TIMESTAMP - INTERVAL '1 hour', NULL),

    -- Failed transfer attempt
    ('d0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000003', 1000.00, 'Large payment attempt', 'IDEMPOT-TRANSFER-005', 'failed', NULL, NULL, CURRENT_TIMESTAMP - INTERVAL '2 days', CURRENT_TIMESTAMP - INTERVAL '2 days', NULL)
ON CONFLICT (idempotency_key) DO NOTHING;

-- Display seed data summary
DO $$
BEGIN
    RAISE NOTICE 'Seed data loaded successfully:';
    RAISE NOTICE '- % users', (SELECT COUNT(*) FROM users WHERE id::text LIKE 'a0000000-%');
    RAISE NOTICE '- % accounts', (SELECT COUNT(*) FROM accounts WHERE id::text LIKE 'b0000000-%');
    RAISE NOTICE '- % transactions', (SELECT COUNT(*) FROM transactions WHERE id::text LIKE 'c0000000-%');
    RAISE NOTICE '- % transfers', (SELECT COUNT(*) FROM transfers WHERE id::text LIKE 'd0000000-%');
END $$;
