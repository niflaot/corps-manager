--liquibase formatted sql
--changeset discord-bot:discord-bot-customers-0002-track-customer-spending
alter table frequent_customers
    add column total_spent bigint not null default 0 check (total_spent >= 0),
    add column last_visit_at timestamptz;

update frequent_customers set last_visit_at = updated_at where last_visit_at is null;

alter table frequent_customers
    alter column last_visit_at set default now(),
    alter column last_visit_at set not null;

create table frequent_customer_visits (
    id bigserial primary key,
    customer_name text not null references frequent_customers(normalized_name) on delete cascade,
    discord_user_id varchar(20) not null check (discord_user_id ~ '^[0-9]{1,20}$'),
    display_name varchar(80) not null,
    amount bigint not null check (amount >= 0),
    attended_at timestamptz not null default now()
);

create index frequent_customer_visits_period_idx
    on frequent_customer_visits (attended_at desc, customer_name);
create index frequent_customer_visits_customer_idx
    on frequent_customer_visits (customer_name, attended_at desc);
--rollback drop table if exists frequent_customer_visits; alter table frequent_customers drop column if exists last_visit_at, drop column if exists total_spent;
