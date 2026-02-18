-- users
INSERT INTO users (id, name, email)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Michael', 'michael@example.com'),
    ('22222222-2222-2222-2222-222222222222', 'Test User', 'test@example.com')
ON CONFLICT (email) DO NOTHING;

-- accounts
INSERT INTO accounts (id, user_id, name, number)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'Main RUB', '40802810000000000001'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', 'Brokerage', '40802810000000000002'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222', 'Wallet', '40802810000000000003')
ON CONFLICT (user_id, number) DO NOTHING;

-- categories
INSERT INTO categories (id, user_id, name)
VALUES
    ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'Salary'),
    ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'Food'),
    ('55555555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 'Transport'),
    ('66666666-6666-6666-6666-666666666666', '22222222-2222-2222-2222-222222222222', 'Entertainment')
ON CONFLICT DO NOTHING;

-- providers
INSERT INTO providers (id, name)
VALUES
    ('77777777-7777-7777-7777-777777777777', 'Tinkoff'),
    ('88888888-8888-8888-8888-888888888888', 'Sberbank'),
    ('99999999-9999-9999-9999-999999999999', 'Yandex Pay')
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
    )
ON CONFLICT (provider_id, external_id) DO NOTHING;
