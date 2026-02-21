-- =============================================================================
-- YOMIRA — 10_USERS
-- Schema: users
-- @author  tai.buivan.jp@gmail.com
-- Tables: account · oauthprovider · session · follow · readingpreference
-- v1.1.0  : readingpreference.datasaver added
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql
-- =============================================================================

-- ============================================================
-- users.account  — Core user record
-- ============================================================
CREATE TABLE users.account (
    id              TEXT            NOT NULL,
    username        VARCHAR(64)     NOT NULL,
    email           VARCHAR(254)    NOT NULL,
    passwordhash    TEXT,
    displayname     VARCHAR(100),
    avatarurl       TEXT,
    bio             TEXT,
    website         TEXT,
    role            VARCHAR(20)     NOT NULL DEFAULT 'member',
    isverified      BOOLEAN         NOT NULL DEFAULT FALSE,
    isactive        BOOLEAN         NOT NULL DEFAULT TRUE,
    lastloginat     TIMESTAMPTZ,
    createdat       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updatedat       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deletedat       TIMESTAMPTZ,

    CONSTRAINT account_pkey        PRIMARY KEY (id),
    CONSTRAINT account_email_uq    UNIQUE (email),
    CONSTRAINT account_username_uq UNIQUE (username)
);
SELECT attach_updatedat_trigger('users', 'account');

CREATE INDEX idx_users_account_email     ON users.account (email)     WHERE deletedat IS NULL;
CREATE INDEX idx_users_account_role      ON users.account (role)      WHERE deletedat IS NULL;
CREATE INDEX idx_users_account_createdat ON users.account (createdat) WHERE deletedat IS NULL;

COMMENT ON TABLE  users.account IS 'Core user record. One row per registered account. Soft-deleted via deletedat.';
COMMENT ON COLUMN users.account.id IS 'UUIDv7 generated at application layer. Time-sortable, globally unique, non-guessable.';
COMMENT ON COLUMN users.account.username IS 'Unique public handle. Restricted to 3–64 chars (validated by Go).';
COMMENT ON COLUMN users.account.email IS 'Primary login identifier. Unique across all accounts.';
COMMENT ON COLUMN users.account.passwordhash IS 'bcrypt hash (cost=12). NULL when user authenticates exclusively via OAuth.';
COMMENT ON COLUMN users.account.role IS 'Permission level. Allowed values (Go): admin | moderator | member | banned.';
COMMENT ON COLUMN users.account.isverified IS 'TRUE after the user confirms their email address.';
COMMENT ON COLUMN users.account.isactive IS 'FALSE = soft-suspended; can login but write actions are blocked.';
COMMENT ON COLUMN users.account.lastloginat IS 'Refreshed on every successful token issuance.';
COMMENT ON COLUMN users.account.deletedat IS 'Non-NULL = soft deleted. Row retained for legal/audit reasons.';

-- ============================================================
-- users.oauthprovider  — External OAuth identity links
-- ============================================================
CREATE TABLE users.oauthprovider (
    id          BIGSERIAL       NOT NULL,
    userid      TEXT            NOT NULL,
    provider    VARCHAR(30)     NOT NULL,
    providerid  TEXT            NOT NULL,
    email       VARCHAR(254),
    accesstoken TEXT,
    createdat   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT oauthprovider_pkey       PRIMARY KEY (id),
    CONSTRAINT oauthprovider_uq         UNIQUE (provider, providerid),
    CONSTRAINT oauthprovider_account_fk FOREIGN KEY (userid) REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('users', 'oauthprovider');

CREATE INDEX idx_users_oauthprovider_userid ON users.oauthprovider (userid);

COMMENT ON TABLE  users.oauthprovider IS 'Links an account to one or more external OAuth providers. One row per (provider, providerid).';
COMMENT ON COLUMN users.oauthprovider.provider IS 'Identity provider name. Allowed values (Go): google | discord | github | apple.';
COMMENT ON COLUMN users.oauthprovider.providerid IS 'Stable unique ID from the provider (e.g. Google "sub" claim). Used as lookup key.';
COMMENT ON COLUMN users.oauthprovider.email IS 'Email returned by the provider. May differ from users.account.email.';
COMMENT ON COLUMN users.oauthprovider.accesstoken IS 'Current OAuth access token. MUST be encrypted at rest in production.';

-- ============================================================
-- users.session  — Active auth sessions (hashed refresh tokens)
-- ============================================================
CREATE TABLE users.session (
    id          BIGSERIAL       NOT NULL,
    userid      TEXT            NOT NULL,
    tokenhash   TEXT            NOT NULL,
    devicename  VARCHAR(200),
    ipaddress   INET,
    useragent   TEXT,
    expiresat   TIMESTAMPTZ     NOT NULL,
    revokedat   TIMESTAMPTZ,
    createdat   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT session_pkey    PRIMARY KEY (id),
    CONSTRAINT session_user_fk FOREIGN KEY (userid) REFERENCES users.account (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_users_session_tokenhash ON users.session (tokenhash);
CREATE INDEX        idx_users_session_userid    ON users.session (userid);
CREATE INDEX        idx_users_session_expiresat ON users.session (expiresat);

COMMENT ON TABLE  users.session IS 'One row per active auth session. Raw bearer tokens are never stored; only SHA-256 hashes.';
COMMENT ON COLUMN users.session.tokenhash IS 'SHA-256 hex digest of the raw bearer token. Primary lookup key in auth middleware.';
COMMENT ON COLUMN users.session.devicename IS 'Human-readable label parsed from User-Agent, e.g. "Chrome on Windows", "iPhone 15".';
COMMENT ON COLUMN users.session.revokedat IS 'Non-NULL = explicitly revoked (logout, password change, admin action).';
COMMENT ON COLUMN users.session.expiresat IS 'Hard expiry. Tokens past this are invalid regardless of revokedat.';

-- ============================================================
-- users.follow  — User-to-user follow graph
-- ============================================================
CREATE TABLE users.follow (
    followerid  TEXT            NOT NULL,
    followingid TEXT            NOT NULL,
    createdat   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT follow_pkey         PRIMARY KEY (followerid, followingid),
    CONSTRAINT follow_follower_fk  FOREIGN KEY (followerid)  REFERENCES users.account (id) ON DELETE CASCADE,
    CONSTRAINT follow_following_fk FOREIGN KEY (followingid) REFERENCES users.account (id) ON DELETE CASCADE
);

CREATE INDEX idx_users_follow_followingid ON users.follow (followingid);

COMMENT ON TABLE  users.follow IS 'Directed follow graph. followerid → followingid. Activity of followingid appears in follower''s Feed.';
COMMENT ON COLUMN users.follow.followerid  IS 'User who initiated the follow.';
COMMENT ON COLUMN users.follow.followingid IS 'User being followed. Self-follow prevented by Go service.';

-- ============================================================
-- users.readingpreference  — Per-user reader UI settings
-- ============================================================
CREATE TABLE users.readingpreference (
    userid          TEXT        NOT NULL,
    readingmode     VARCHAR(20) NOT NULL DEFAULT 'ltr',
    pagefit         VARCHAR(20) NOT NULL DEFAULT 'width',
    doublepageon    BOOLEAN     NOT NULL DEFAULT FALSE,
    showpagebar     BOOLEAN     NOT NULL DEFAULT TRUE,
    preloadpages    SMALLINT    NOT NULL DEFAULT 3,
    datasaver       BOOLEAN     NOT NULL DEFAULT FALSE,
    hidensfw        BOOLEAN     NOT NULL DEFAULT TRUE,
    hidelanguages   TEXT[]      NOT NULL DEFAULT '{}',
    createdat       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT readingpref_pkey    PRIMARY KEY (userid),
    CONSTRAINT readingpref_user_fk FOREIGN KEY (userid) REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('users', 'readingpreference');

COMMENT ON TABLE  users.readingpreference IS '1:1 with users.account. Stores reader UI preferences. Created on first save, not at registration.';
COMMENT ON COLUMN users.readingpreference.readingmode IS 'Chapter layout. Allowed values (Go): ltr | rtl | vertical | webtoon.';
COMMENT ON COLUMN users.readingpreference.pagefit IS 'Image scaling. Allowed values (Go): width | height | original | stretch.';
COMMENT ON COLUMN users.readingpreference.preloadpages IS 'Pages to preload ahead. Range 1–10 enforced by Go service.';
COMMENT ON COLUMN users.readingpreference.datasaver IS 'TRUE = serve compressed/low-res images (Data Saver mode). Reduces bandwidth up to 70%.';
COMMENT ON COLUMN users.readingpreference.hidensfw IS 'TRUE = explicit/suggestive content hidden from all listings.';
COMMENT ON COLUMN users.readingpreference.hidelanguages IS 'Array of BCP-47 codes. Chapters in these languages are hidden, e.g. {''en'',''ja''}.';

-- =============================================================================
-- END OF 10_USERS
-- =============================================================================
