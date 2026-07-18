--liquibase formatted sql
--changeset discord-bot:discord-bot-settings-0001-create-settings
create table settings (
    key text primary key check (key ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    value jsonb not null,
    revision bigint not null default 1 check (revision > 0),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
--rollback drop table if exists settings;
