--liquibase formatted sql
--changeset discord-bot:discord-bot-inactivity-0001-create-inactivity-dismissals
create table inactivity_dismissals (
    normalized_name text primary key,
    display_name text not null,
    added_by text not null,
    added_at timestamptz not null default now(),
    constraint inactivity_dismissals_normalized_name_format
        check (normalized_name ~ '^[a-z]+_[a-z]+$'),
    constraint inactivity_dismissals_display_name_format
        check (display_name ~ '^[A-Za-z]+_[A-Za-z]+$')
);

--rollback drop table if exists inactivity_dismissals;
