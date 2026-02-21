-- =============================================================================
-- YOMIRA — 70_SYSTEM
-- Schema: system
-- @author  tai.buivan.jp@gmail.com
-- Tables: auditlog · setting · announcement
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql · 10_USERS/USERS.sql
-- =============================================================================

-- ============================================================
-- system.auditlog  — Immutable record of privileged actions
-- ============================================================
-- Append-only. Never UPDATE or DELETE rows.
-- No updatedat column by design.
-- ============================================================
CREATE TABLE system.auditlog (
    id          TEXT         NOT NULL,
    actorid     TEXT         NOT NULL,
    action      VARCHAR(100) NOT NULL,
    entitytype  VARCHAR(50),
    entityid    TEXT,
    before      JSONB,
    after       JSONB,
    ipaddress   INET,
    createdat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT auditlog_pkey     PRIMARY KEY (id),
    CONSTRAINT auditlog_actor_fk FOREIGN KEY (actorid) REFERENCES users.account (id) ON DELETE SET NULL
);

CREATE INDEX idx_system_auditlog_actorid   ON system.auditlog (actorid);
CREATE INDEX idx_system_auditlog_action    ON system.auditlog (action);
CREATE INDEX idx_system_auditlog_createdat ON system.auditlog (createdat);

COMMENT ON TABLE  system.auditlog IS 'Immutable record of all privileged administrative actions. Append-only: never UPDATE or DELETE rows. actorid uses ON DELETE SET NULL to preserve audit trail even if admin is deleted.';
COMMENT ON COLUMN system.auditlog.id IS 'UUIDv7. Time-ordered; newest records sort to end of B-tree naturally.';
COMMENT ON COLUMN system.auditlog.action IS 'Dot-notation action name, e.g. "comic.delete", "user.ban", "chapter.lock".';
COMMENT ON COLUMN system.auditlog.before IS 'JSON snapshot of entity state BEFORE the action. NULL for create operations.';
COMMENT ON COLUMN system.auditlog.after IS 'JSON snapshot of entity state AFTER the action. NULL for delete operations.';

-- ============================================================
-- system.setting  — Global key-value configuration store
-- ============================================================
CREATE SEQUENCE system.setting_id_seq START 100000 INCREMENT 1;

CREATE TABLE system.setting (
    id          INTEGER      NOT NULL DEFAULT nextval('system.setting_id_seq'),
    key         VARCHAR(100) NOT NULL,
    value       TEXT,
    description TEXT,
    createdat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT setting_pkey   PRIMARY KEY (id),
    CONSTRAINT setting_key_uq UNIQUE (key)
);
SELECT attach_updatedat_trigger('system', 'setting');

COMMENT ON TABLE  system.setting IS 'Global key-value configuration. Read by Go app at startup and cached in memory. Use dot-notation keys for namespacing.';
COMMENT ON COLUMN system.setting.key IS 'Dot-notation namespaced key, e.g. "site.name", "crawler.max_retries", "auth.session_ttl_days".';
COMMENT ON COLUMN system.setting.value IS 'String value. Go service is responsible for type coercion (bool, int, duration, etc.).';
COMMENT ON COLUMN system.setting.description IS 'Human-readable explanation shown in the admin settings UI.';

-- ============================================================
-- system.announcement  — Site-wide announcements
-- ============================================================
CREATE TABLE system.announcement (
    id          TEXT         NOT NULL,
    authorid    TEXT         NOT NULL,
    title       VARCHAR(300) NOT NULL,
    body        TEXT         NOT NULL,
    bodyformat  VARCHAR(10)  NOT NULL DEFAULT 'markdown',
    ispublished BOOLEAN      NOT NULL DEFAULT FALSE,
    ispinned    BOOLEAN      NOT NULL DEFAULT FALSE,
    publishedat TIMESTAMPTZ,
    expiresat   TIMESTAMPTZ,
    createdat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deletedat   TIMESTAMPTZ,

    CONSTRAINT announcement_pkey      PRIMARY KEY (id),
    CONSTRAINT announcement_author_fk FOREIGN KEY (authorid) REFERENCES users.account (id) ON DELETE SET NULL
);
SELECT attach_updatedat_trigger('system', 'announcement');

CREATE INDEX idx_system_announcement_published
    ON system.announcement (publishedat DESC)
    WHERE ispublished = TRUE AND deletedat IS NULL;

COMMENT ON TABLE  system.announcement IS 'Site-wide announcements published by admins or moderators. Shown on the Community / Announcements page. Soft-deleted via deletedat.';
COMMENT ON COLUMN system.announcement.bodyformat IS 'Rendering format. Allowed values (Go): markdown | html | plain.';
COMMENT ON COLUMN system.announcement.ispinned IS 'TRUE = shown at the very top of the announcements list regardless of date.';
COMMENT ON COLUMN system.announcement.publishedat IS 'When the announcement became visible. NULL = draft (not yet published).';
COMMENT ON COLUMN system.announcement.expiresat IS 'NULL = never expires. Non-NULL = auto-hidden after this timestamp by Go.';

-- =============================================================================
-- END OF 70_SYSTEM
-- =============================================================================
