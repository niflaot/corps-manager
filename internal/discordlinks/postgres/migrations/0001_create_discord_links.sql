--liquibase formatted sql
--changeset discord-bot:discord-bot-links-0001-create-discord-links
create table discord_links (
    id uuid primary key default gen_random_uuid(),
    subject text not null check (subject ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    discord_user_id text not null check (discord_user_id ~ '^[0-9]{1,20}$'),
    username text not null check (length(username) between 1 and 64),
    global_name text not null default '' check (length(global_name) <= 64),
    avatar_hash text not null default '' check (length(avatar_hash) <= 128),
    scopes text[] not null default array[]::text[],
    linked_at timestamptz not null,
    unlinked_at timestamptz,
    created_at timestamptz not null,
    updated_at timestamptz not null,
    check (unlinked_at is null or unlinked_at >= linked_at)
);
create unique index discord_links_active_subject_uq on discord_links(subject) where unlinked_at is null;
create unique index discord_links_active_user_uq on discord_links(discord_user_id) where unlinked_at is null;
create index discord_links_subject_history_idx on discord_links(subject, created_at desc);

create table discord_link_intents (
    id uuid primary key default gen_random_uuid(),
    kind text not null check (kind in ('link', 'login')),
    subject text check (subject ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    completion_key text not null check (completion_key ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    idempotency_key text not null unique check (idempotency_key ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    request_hash char(64) not null,
    status text not null default 'pending' check (status in ('pending', 'started', 'processing', 'completed')),
    state_hash char(64) unique,
    result_hash char(64) unique,
    result_status text check (result_status in ('linked', 'authenticated', 'not_linked', 'denied', 'conflict', 'failed')),
    result_error_code text,
    result_expires_at timestamptz,
    result_consumed_at timestamptz,
    result_consumer_key text,
    link_id uuid references discord_links(id) on delete restrict,
    expires_at timestamptz not null,
    started_at timestamptz,
    completed_at timestamptz,
    created_at timestamptz not null,
    updated_at timestamptz not null,
    check (result_consumed_at is null or result_consumer_key is not null),
    check ((kind = 'link' and subject is not null) or (kind = 'login' and subject is null))
);
create index discord_link_intents_expiry_idx on discord_link_intents(expires_at);
create index discord_link_results_expiry_idx on discord_link_intents(result_expires_at) where result_hash is not null;
create index discord_link_intents_updated_idx on discord_link_intents(updated_at);
--rollback drop table if exists discord_link_intents; drop table if exists discord_links;
