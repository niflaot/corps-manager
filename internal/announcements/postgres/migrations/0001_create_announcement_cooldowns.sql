--liquibase formatted sql

--changeset corps-manager:corps-manager-announcements-0001-create-announcement-cooldowns
CREATE TABLE announcement_cooldowns (
    cooldown_key TEXT PRIMARY KEY,
    actor TEXT NOT NULL,
    announced_at TIMESTAMPTZ NOT NULL,
    available_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT announcement_cooldowns_window CHECK (available_at > announced_at)
);

--rollback DROP TABLE IF EXISTS announcement_cooldowns;
