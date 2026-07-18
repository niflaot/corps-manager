--liquibase formatted sql
--changeset discord-bot:discord-bot-messages-0001-create-managed-messages
create table managed_messages (
    id uuid primary key default gen_random_uuid(),
    key text not null unique,
    guild_id text not null,
    channel_id text not null,
    discord_message_id text,
    payload jsonb not null check (jsonb_typeof(payload) = 'object'),
    desired_hash char(64) not null,
    observed_hash char(64),
    revision bigint not null default 1 check (revision > 0),
    state text not null default 'pending' check (state in ('pending', 'healthy', 'drifted', 'repairing', 'blocked', 'archived')),
    failure_count integer not null default 0 check (failure_count >= 0),
    next_check_at timestamptz not null default now(),
    lease_owner text,
    lease_until timestamptz,
    last_checked_at timestamptz,
    last_repaired_at timestamptz,
    last_error text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create unique index managed_messages_remote_id_uq on managed_messages (channel_id, discord_message_id) where discord_message_id is not null;
create index managed_messages_due_idx on managed_messages (state, next_check_at) where state not in ('archived', 'blocked');
create index managed_messages_lease_idx on managed_messages (lease_until) where lease_until is not null;

create table message_idempotency (
    idempotency_key text primary key,
    operation text not null,
    request_hash char(64) not null,
    status_code integer,
    response jsonb,
    expires_at timestamptz not null default now() + interval '24 hours',
    created_at timestamptz not null default now()
);
create index message_idempotency_expiry_idx on message_idempotency (expires_at);
--rollback drop table if exists message_idempotency; drop table if exists managed_messages;
