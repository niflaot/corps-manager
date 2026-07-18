--liquibase formatted sql
--changeset discord-bot:discord-bot-messages-0002-require-components-v2
alter table managed_messages add constraint managed_messages_components_v2_ck check (
    jsonb_exists(payload, 'components')
    and jsonb_typeof(payload->'components') = 'array'
    and not jsonb_exists(payload, 'content')
    and not jsonb_exists(payload, 'embeds')
);
--rollback alter table managed_messages drop constraint if exists managed_messages_components_v2_ck;
