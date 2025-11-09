-- Insert test data for development and testing
-- Creates sample accounts with various balances

-- Account 1: Standard account with 1000 RUB
INSERT INTO accounts (id, balance_value, balance_currency_code, created_at, updated_at)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 1000.00, 'RUB', NOW(), NOW());

-- Account 2: Standard account with 500 RUB
INSERT INTO accounts (id, balance_value, balance_currency_code, created_at, updated_at)
VALUES 
    ('22222222-2222-2222-2222-222222222222', 500.00, 'RUB', NOW(), NOW());

-- Account 3: High balance account with 10000 RUB
INSERT INTO accounts (id, balance_value, balance_currency_code, created_at, updated_at)
VALUES 
    ('33333333-3333-3333-3333-333333333333', 10000.00, 'RUB', NOW(), NOW());

-- Account 4: Low balance account with 50 RUB
INSERT INTO accounts (id, balance_value, balance_currency_code, created_at, updated_at)
VALUES 
    ('44444444-4444-4444-4444-444444444444', 50.00, 'RUB', NOW(), NOW());

-- Account 5: Zero balance account
INSERT INTO accounts (id, balance_value, balance_currency_code, created_at, updated_at)
VALUES 
    ('55555555-5555-5555-5555-555555555555', 0.00, 'RUB', NOW(), NOW());

-- Note: These are test accounts for development purposes
-- In production, use a separate data seeding strategy
