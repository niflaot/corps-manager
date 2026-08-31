--liquibase formatted sql
--changeset discord-bot:discord-bot-customers-0001-create-frequent-customers
create table frequent_customers (
    normalized_name text primary key check (normalized_name ~ '^[[:alnum:]][[:alnum:]_]{0,63}$'),
    visits bigint not null default 0 check (visits >= 0),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table frequent_customer_attendants (
    customer_name text not null references frequent_customers(normalized_name) on delete cascade,
    discord_user_id varchar(20) not null check (discord_user_id ~ '^[0-9]{1,20}$'),
    display_name varchar(80) not null,
    visits bigint not null default 0 check (visits >= 0),
    first_attended_at timestamptz not null default now(),
    last_attended_at timestamptz not null default now(),
    primary key (customer_name, discord_user_id)
);

create index frequent_customer_attendants_customer_visits_idx
    on frequent_customer_attendants (customer_name, visits desc);
--rollback drop table if exists frequent_customer_attendants; drop table if exists frequent_customers;
