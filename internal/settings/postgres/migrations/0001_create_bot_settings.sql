--liquibase formatted sql
--changeset discord-bot:discord-bot-settings-0001-create-bot-settings
create table bot_settings (
    setting_key varchar(96) primary key check (setting_key ~ '^[a-z0-9][a-z0-9_.-]{0,95}$'),
    setting_value text not null check (length(setting_value) <= 1024),
    updated_at timestamptz not null default now()
);
--rollback drop table if exists bot_settings;
