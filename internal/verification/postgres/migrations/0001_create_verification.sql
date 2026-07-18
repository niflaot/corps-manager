--liquibase formatted sql
--changeset discord-bot:discord-bot-verification-0001-create-verification
create table verification_groups (
    id uuid primary key default gen_random_uuid(),
    key text not null unique check (key ~ '^[a-z0-9][a-z0-9_-]{0,31}$'),
    role_id text not null,
    button_label text not null check (length(button_label) between 1 and 80),
    button_emoji text not null default '',
    button_style smallint not null default 1 check (button_style between 1 and 4),
    position smallint not null unique check (position between 1 and 5),
    enabled boolean not null default true,
    revision bigint not null default 1 check (revision > 0),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table verification_memberships (
    id uuid primary key default gen_random_uuid(),
    user_id text not null,
    group_id uuid not null references verification_groups(id) on delete restrict,
    role_id text not null,
    verified_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique(user_id, group_id)
);
create index verification_memberships_user_idx on verification_memberships(user_id);
--rollback drop table if exists verification_memberships; drop table if exists verification_groups;
