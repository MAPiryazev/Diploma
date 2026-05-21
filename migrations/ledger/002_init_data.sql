-- users
INSERT INTO users (id, name, email)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Michael', 'michael@example.com'),
    ('22222222-2222-2222-2222-222222222222', 'Test User', 'test@example.com'),
    ('aaaaaaaa-4444-4444-4444-444444444444', 'Ivan', 'ivan@example.com'),
    ('aaaaaaaa-5555-5555-5555-555555555555', 'Maria', 'maria@example.com')
ON CONFLICT (email) DO NOTHING;

-- accounts
INSERT INTO accounts (id, user_id, name, number)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'Main RUB', '40802810000000000001'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', 'Brokerage', '40802810000000000002'),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '11111111-1111-1111-1111-111111111111', 'Savings', '40802810000000000004'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222', 'Wallet', '40802810000000000003'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'aaaaaaaa-4444-4444-4444-444444444444', 'Main RUB', '40802810000000000005'),
    ('ffffffff-ffff-ffff-ffff-ffffffffffff', 'aaaaaaaa-4444-4444-4444-444444444444', 'EUR pocket', '40802810000000000006'),
    ('12121212-1212-1212-1212-121212121212', 'aaaaaaaa-5555-5555-5555-555555555555', 'Debit', '40802810000000000007')
ON CONFLICT (user_id, number) DO NOTHING;

-- categories
INSERT INTO categories (id, user_id, name)
VALUES
    ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'Salary'),
    ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'Food'),
    ('55555555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 'Transport'),
    ('aaaaaa01-aaaa-aaaa-aaaa-aaaaaaaaaa01', '11111111-1111-1111-1111-111111111111', 'Utilities'),
    ('aaaaaa02-aaaa-aaaa-aaaa-aaaaaaaaaa02', '11111111-1111-1111-1111-111111111111', 'Subscriptions'),
    ('aaaaaa03-aaaa-aaaa-aaaa-aaaaaaaaaa03', '11111111-1111-1111-1111-111111111111', 'Health'),
    ('66666666-6666-6666-6666-666666666666', '22222222-2222-2222-2222-222222222222', 'Entertainment'),
    ('bbbbbb01-bbbb-bbbb-bbbb-bbbbbbbbbb01', '22222222-2222-2222-2222-222222222222', 'Shopping'),
    ('cccccc01-cccc-cccc-cccc-cccccccccc01', 'aaaaaaaa-4444-4444-4444-444444444444', 'Freelance'),
    ('cccccc02-cccc-cccc-cccc-cccccccccc02', 'aaaaaaaa-5555-5555-5555-555555555555', 'Gifts')
ON CONFLICT DO NOTHING;

-- providers
INSERT INTO providers (id, name)
VALUES
    ('77777777-7777-7777-7777-777777777777', 'Tinkoff'),
    ('88888888-8888-8888-8888-888888888888', 'Sberbank'),
    ('99999999-9999-9999-9999-999999999999', 'Yandex Pay'),
    ('a0a0a0a0-0a0a-0a0a-0a0a-0a0a0a0a0a0a', 'Alfa Bank'),
    ('b0b0b0b0-0b0b-0b0b-0b0b-0b0b0b0b0b0b', 'Ozon Pay')
ON CONFLICT (name) DO NOTHING;

-- transactions
INSERT INTO transactions (
    id,
    user_id,
    amount,
    currency,
    from_account_id,
    to_account_id,
    provider_id,
    category_id,
    type,
    status,
    description,
    external_id,
    occurred_at
)
VALUES
    (
        'aaaaaaaa-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        150000.00,
        'RUB',
        NULL,
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        '77777777-7777-7777-7777-777777777777',
        '33333333-3333-3333-3333-333333333333',
        'income',
        'done',
        'Salary for December',
        'ext-1',
        NOW() - INTERVAL '25 days'
    ),
    (
        'bbbbbbbb-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        1200.50,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        NULL,
        '88888888-8888-8888-8888-888888888888',
        '44444444-4444-4444-4444-444444444444',
        'expense',
        'done',
        'Groceries at supermarket',
        'ext-2',
        NOW() - INTERVAL '5 days'
    ),
    (
        'cccccccc-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        300.00,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        NULL,
        '88888888-8888-8888-8888-888888888888',
        '55555555-5555-5555-5555-555555555555',
        'expense',
        'done',
        'Metro top-up',
        'ext-3',
        NOW() - INTERVAL '3 days'
    ),
    (
        'dddddddd-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        5000.00,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        '77777777-7777-7777-7777-777777777777',
        NULL,
        'transfer',
        'done',
        'Transfer to brokerage account',
        'ext-4',
        NOW() - INTERVAL '10 days'
    ),
    (
        'eeeeeeee-1111-1111-1111-111111111111',
        '22222222-2222-2222-2222-222222222222',
        2500.00,
        'RUB',
        NULL,
        'cccccccc-cccc-cccc-cccc-cccccccccccc',
        '99999999-9999-9999-9999-999999999999',
        '66666666-6666-6666-6666-666666666666',
        'income',
        'pending',
        'Bonus from side project',
        'ext-5',
        NOW() - INTERVAL '1 days'
    ),
    (
        'f1f1f1f1-1111-1111-1111-111111111111',
        '11111111-1111-1111-1111-111111111111',
        450.00,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        NULL,
        '88888888-8888-8888-8888-888888888888',
        '44444444-4444-4444-4444-444444444444',
        'expense',
        'done',
        'Coffee and breakfast',
        'ext-6',
        NOW() - INTERVAL '2 days'
    ),
    (
        'f1f1f1f1-2222-2222-2222-222222222222',
        '11111111-1111-1111-1111-111111111111',
        899.00,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        NULL,
        '77777777-7777-7777-7777-777777777777',
        'aaaaaa02-aaaa-aaaa-aaaa-aaaaaaaaaa02',
        'expense',
        'done',
        'Streaming subscription',
        'ext-7',
        NOW() - INTERVAL '7 days'
    ),
    (
        'f1f1f1f1-3333-3333-3333-333333333333',
        '11111111-1111-1111-1111-111111111111',
        500.00,
        'RUB',
        NULL,
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        '77777777-7777-7777-7777-777777777777',
        '33333333-3333-3333-3333-333333333333',
        'income',
        'done',
        'Cashback promo',
        'ext-8',
        NOW() - INTERVAL '12 days'
    ),
    (
        'f1f1f1f1-4444-4444-4444-444444444444',
        '11111111-1111-1111-1111-111111111111',
        10000.00,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        'dddddddd-dddd-dddd-dddd-dddddddddddd',
        '77777777-7777-7777-7777-777777777777',
        NULL,
        'transfer',
        'done',
        'To savings',
        'ext-9',
        NOW() - INTERVAL '15 days'
    ),
    (
        'f1f1f1f1-5555-5555-5555-555555555555',
        '11111111-1111-1111-1111-111111111111',
        3200.00,
        'RUB',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        NULL,
        '88888888-8888-8888-8888-888888888888',
        'aaaaaa01-aaaa-aaaa-aaaa-aaaaaaaaaa01',
        'expense',
        'failed',
        'Card declined at utility payment',
        'ext-10',
        NOW() - INTERVAL '4 days'
    ),
    (
        'f1f1f1f1-6666-6666-6666-666666666666',
        '11111111-1111-1111-1111-111111111111',
        75000.00,
        'RUB',
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        '77777777-7777-7777-7777-777777777777',
        NULL,
        'transfer',
        'pending',
        'Brokerage withdrawal pending',
        'ext-11',
        NOW() - INTERVAL '6 hours'
    ),
    (
        'f2f2f2f2-1111-1111-1111-111111111111',
        '22222222-2222-2222-2222-222222222222',
        890.00,
        'RUB',
        'cccccccc-cccc-cccc-cccc-cccccccccccc',
        NULL,
        '99999999-9999-9999-9999-999999999999',
        'bbbbbb01-bbbb-bbbb-bbbb-bbbbbbbbbb01',
        'expense',
        'done',
        'Online marketplace',
        'ext-12',
        NOW() - INTERVAL '8 days'
    ),
    (
        'f2f2f2f2-2222-2222-2222-222222222222',
        '22222222-2222-2222-2222-222222222222',
        12000.00,
        'RUB',
        NULL,
        'cccccccc-cccc-cccc-cccc-cccccccccccc',
        '88888888-8888-8888-8888-888888888888',
        '66666666-6666-6666-6666-666666666666',
        'income',
        'done',
        'Freelance invoice paid',
        'ext-13',
        NOW() - INTERVAL '18 days'
    ),
    (
        'f3f3f3f3-1111-1111-1111-111111111111',
        'aaaaaaaa-4444-4444-4444-444444444444',
        45000.00,
        'RUB',
        NULL,
        'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
        'a0a0a0a0-0a0a-0a0a-0a0a-0a0a0a0a0a0a',
        'cccccc01-cccc-cccc-cccc-cccccccccc01',
        'income',
        'done',
        'Invoice #1042',
        'ext-14',
        NOW() - INTERVAL '20 days'
    ),
    (
        'f3f3f3f3-2222-2222-2222-222222222222',
        'aaaaaaaa-4444-4444-4444-444444444444',
        120.50,
        'EUR',
        'ffffffff-ffff-ffff-ffff-ffffffffffff',
        NULL,
        'a0a0a0a0-0a0a-0a0a-0a0a-0a0a0a0a0a0a',
        'cccccc01-cccc-cccc-cccc-cccccccccc01',
        'expense',
        'done',
        'Software license EUR',
        'ext-15',
        NOW() - INTERVAL '9 days'
    ),
    (
        'f3f3f3f3-3333-3333-3333-333333333333',
        'aaaaaaaa-4444-4444-4444-444444444444',
        5000.00,
        'RUB',
        'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
        'ffffffff-ffff-ffff-ffff-ffffffffffff',
        '77777777-7777-7777-7777-777777777777',
        NULL,
        'transfer',
        'done',
        'EUR pocket top-up',
        'ext-16',
        NOW() - INTERVAL '11 days'
    ),
    (
        'f4f4f4f4-1111-1111-1111-111111111111',
        'aaaaaaaa-5555-5555-5555-555555555555',
        3500.00,
        'RUB',
        NULL,
        '12121212-1212-1212-1212-121212121212',
        'b0b0b0b0-0b0b-0b0b-0b0b-0b0b0b0b0b0b',
        'cccccc02-cccc-cccc-cccc-cccccccccc02',
        'income',
        'pending',
        'Gift money from family',
        'ext-17',
        NOW() - INTERVAL '3 hours'
    ),
    (
        'f4f4f4f4-2222-2222-2222-222222222222',
        'aaaaaaaa-5555-5555-5555-555555555555',
        199.00,
        'RUB',
        '12121212-1212-1212-1212-121212121212',
        NULL,
        '99999999-9999-9999-9999-999999999999',
        'cccccc02-cccc-cccc-cccc-cccccccccc02',
        'expense',
        'done',
        'Flowers',
        'ext-18',
        NOW() - INTERVAL '14 days'
    )
ON CONFLICT (provider_id, external_id) DO NOTHING;
