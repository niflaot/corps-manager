--liquibase formatted sql
--changeset discord-bot:discord-bot-messages-0003-timestamp-idempotency
alter table message_idempotency add column updated_at timestamptz not null default now();
--rollback alter table message_idempotency drop column if exists updated_at;
