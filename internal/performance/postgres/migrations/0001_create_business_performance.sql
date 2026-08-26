--liquibase formatted sql
--changeset discord-bot:discord-bot-performance-0001-create-business-performance
create table business_performance_state (
    business_id bigint primary key check (business_id > 0),
    state jsonb not null check (jsonb_typeof(state) = 'object'),
    revision bigint not null default 1 check (revision > 0),
    updated_at timestamptz not null default now()
);
--rollback drop table if exists business_performance_state;
