--liquibase formatted sql
--changeset discord-bot:discord-bot-agreements-0001-create-business-agreements
create table business_agreements (
    agreement_id varchar(64) primary key check (agreement_id ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    description varchar(1000) not null check (length(description) >= 3),
    image_url text,
    created_by varchar(20) not null check (created_by ~ '^[0-9]{1,20}$'),
    created_at timestamptz not null default now()
);
--rollback drop table if exists business_agreements;
