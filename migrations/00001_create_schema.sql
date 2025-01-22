DO
$$
BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
CREATE DOMAIN UINT256 AS NUMERIC
    CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
ELSE
ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
ALTER DOMAIN UINT256 ADD
    CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
END IF;
END
$$;


create table if not exists  business
(
    guid varchar primary key,
    business_uid varchar not null,
    notify_url varchar not null,
    timestamp bigint not null check ( timestamp > 0 )
);
create index if not exists tokens_timestamp on business(timestamp);
create unique index if not exists business_ui on business (business_uid);


create table if not exists blocks
(
    hash varchar primary key,
    parent_hash varchar not null unique,
    number UINT256 not null unique check(number > 0),
    timestamp bigint not null check(timestamp > 0)
);
CREATE INDEX IF NOT EXISTS blocks_number ON blocks (number);
CREATE INDEX IF NOT EXISTS blocks_timestamp ON blocks (timestamp);

create table if not exists transactions
(
    guid varchar primary key ,
    block_hash varchar not null,
    block_number uint256 not null check ( block_number > 0 ),
    hash varchar not null,
    from_address varchar not null,
    to_address varchar not null,
    token_address varchar not null,
    token_id varchar not null,
    token_meta varchar not null,
    fee uint256 not null,
    amount uint256 not null,
    status varchar not null,
    tx_type varchar not null,
    timestamp bigint not null check ( timestamp > 0 )
);
CREATE INDEX IF NOT EXISTS transactions_hash ON transactions (hash);
CREATE INDEX IF NOT EXISTS transactions_timestamp ON transactions (timestamp);


--  addresses
create table if not exists addresses
(
    guid varchar primary key ,
    address varchar unique not null,
    address_type varchar(10) not null default 'eoa',
    public_key varchar not null,
    timestamp bigint not null,
    constraint check_timestamp check ( timestamp > 0 ),
    constraint check_address_type check ( address_type in ('eoa', 'hot', 'cold') )
);
CREATE INDEX IF NOT EXISTS idx_addresses_address ON addresses (address);
CREATE INDEX IF NOT EXISTS idx_addresses_address_type ON addresses (address_type);



-- tokens
create table if not exists tokens
(
    guid varchar primary key ,
    token_address varchar not null,
    decimals smallint not null default 18,
    token_name varchar not null,
    collect_amount uint256 not null,
    timestamp bigint not null check ( timestamp > 0 )
);
CREATE INDEX IF NOT EXISTS tokens_timestamp ON tokens (timestamp);
CREATE INDEX IF NOT EXISTS tokens_token_address ON tokens (token_address);


-- banlances
create table if not exists balances
(
    guid varchar primary key ,
    address varchar not null,
    token_address varchar not null,
    address_type varchar(10) not null default 'eoa',
    balance uint256 not null default 0 check ( balance >= 0 ),
    lock_balance uint256 not null default 0,
    timestamp bigint not null,
    CONSTRAINT check_timestamp CHECK (timestamp > 0),
    CONSTRAINT check_address_type CHECK (address_type IN ('eoa', 'hot', 'cold'))
);
CREATE INDEX IF NOT EXISTS idx_balances_address ON balances (address);
CREATE INDEX IF NOT EXISTS idx_balances_token_address ON balances (token_address);
CREATE INDEX IF NOT EXISTS idx_balances_address_type ON balances (address_type);


-- deposits
create table if not exists deposits
(
    guid varchar primary key ,
    timestamp bigint not null check ( timestamp > 0 ),
    status varchar not null,
    confirms smallint not null default 0,

    block_hash varchar not null,
    block_number uint256 not null check ( block_number > 0 ),
    hash varchar not null,
    tx_type varchar not null,

    from_address varchar not null,
    to_address varchar not null,
    amount uint256 not null,

    gas_limit integer not null,
    max_fee_per_gas varchar not null,
    max_priority_fee_per_gas varchar not null ,

    token_type varchar not null ,
    token_address varchar not null ,
    token_id varchar not null ,
    token_meta varchar not null ,

    tx_sign_hex varchar not null
);
CREATE INDEX IF NOT EXISTS deposits_hash ON deposits (hash);
CREATE INDEX IF NOT EXISTS deposits_timestamp ON deposits (timestamp);
CREATE INDEX IF NOT EXISTS deposits_from_address ON deposits (from_address);
CREATE INDEX IF NOT EXISTS deposits_to_address ON deposits (to_address);


create table if not exists withdraws
(
    guid varchar primary key ,
    timestamp bigint not null check ( timestamp > 0 ),
    status varchar not null,

    block_hash varchar not null,
    block_number uint256 not null check ( block_number > 0 ),
    hash varchar not null,
    tx_type varchar not null,

    from_address varchar not null,
    to_address varchar not null,
    amount uint256 not null,

    gas_limit integer not null,
    max_fee_per_gas varchar not null,
    max_priority_fee_per_gas varchar not null ,

    token_type varchar not null ,
    token_address varchar not null ,
    token_id varchar not null ,
    token_meta varchar not null ,

    tx_sign_hex varchar not null
);
CREATE INDEX IF NOT EXISTS deposits_hash ON withdraws (hash);
CREATE INDEX IF NOT EXISTS deposits_timestamp ON withdraws (timestamp);
CREATE INDEX IF NOT EXISTS deposits_from_address ON withdraws (from_address);
CREATE INDEX IF NOT EXISTS deposits_to_address ON withdraws (to_address);



create table if not exists internals
(
    guid varchar primary key ,
    timestamp bigint not null check ( timestamp > 0 ),
    status varchar not null,

    block_hash varchar not null,
    block_number uint256 not null check ( block_number > 0 ),
    hash varchar not null,
    tx_type varchar not null,

    from_address varchar not null,
    to_address varchar not null,
    amount uint256 not null,

    gas_limit integer not null,
    max_fee_per_gas varchar not null,
    max_priority_fee_per_gas varchar not null ,

    token_type varchar not null ,
    token_address varchar not null ,
    token_id varchar not null ,
    token_meta varchar not null ,

    tx_sign_hex varchar not null
);
CREATE INDEX IF NOT EXISTS deposits_hash ON internals (hash);
CREATE INDEX IF NOT EXISTS deposits_timestamp ON internals (timestamp);
CREATE INDEX IF NOT EXISTS deposits_from_address ON internals (from_address);
CREATE INDEX IF NOT EXISTS deposits_to_address ON internals (to_address);