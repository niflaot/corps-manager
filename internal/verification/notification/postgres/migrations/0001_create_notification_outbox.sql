--liquibase formatted sql
--changeset discord-bot:discord-bot-verification-notification-0001-create-outbox
create table verification_notification_outbox (
    id uuid primary key default gen_random_uuid(),
    idempotency_key text not null unique,
    kind text not null check (kind in ('verified', 'unverified')),
    user_id text not null,
    group_id text not null,
    group_key text not null,
    state text not null default 'pending' check (state in ('pending', 'delivering', 'retry', 'delivered', 'dead')),
    attempts integer not null default 0 check (attempts >= 0),
    next_attempt_at timestamptz not null default now(),
    lease_owner text,
    lease_until timestamptz,
    discord_message_id text,
    delivered_at timestamptz,
    last_error text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create index verification_notification_outbox_due_idx
    on verification_notification_outbox (state, next_attempt_at)
    where state in ('pending', 'retry', 'delivering');
create index verification_notification_outbox_dead_idx
    on verification_notification_outbox (updated_at)
    where state = 'dead';
--rollback drop table if exists verification_notification_outbox;
