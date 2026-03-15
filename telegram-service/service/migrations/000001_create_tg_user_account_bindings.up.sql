CREATE TABLE IF NOT EXISTS tg_user_account_bindings (
    telegram_id BIGINT PRIMARY KEY,
    account_id  UUID   NOT NULL
);
