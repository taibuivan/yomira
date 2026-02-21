-- =============================================================================
-- YOMIRA — 30_LIBRARY
-- Schema: library
-- @author  tai.buivan.jp@gmail.com
-- Tables: entry · customlist · customlistitem · readingprogress · chapterread · viewhistory
-- v1.1.0  : viewhistory added; idx_library_entry_updatedat added
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql · 10_USERS/USERS.sql · 20_CORE/CORE.sql
-- =============================================================================

-- ============================================================
-- library.entry  — User's personal comic shelf
-- ============================================================
CREATE TABLE library.entry (
    id                  BIGSERIAL   NOT NULL,
    userid              TEXT        NOT NULL,
    comicid             TEXT        NOT NULL,
    readingstatus       VARCHAR(20) NOT NULL DEFAULT 'plan_to_read',
    score               SMALLINT,
    hasnew              BOOLEAN     NOT NULL DEFAULT FALSE,
    notes               TEXT,
    lastreadchapterid   TEXT,
    lastreadat          TIMESTAMPTZ,
    createdat           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT entry_pkey     PRIMARY KEY (id),
    CONSTRAINT entry_uq       UNIQUE (userid, comicid),
    CONSTRAINT entry_user_fk  FOREIGN KEY (userid)  REFERENCES users.account (id) ON DELETE CASCADE,
    CONSTRAINT entry_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic    (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('library', 'entry');

CREATE INDEX idx_library_entry_userid    ON library.entry (userid, readingstatus);
CREATE INDEX idx_library_entry_comicid  ON library.entry (comicid);
CREATE INDEX idx_library_entry_hasnew   ON library.entry (userid)    WHERE hasnew = TRUE;
CREATE INDEX idx_library_entry_updatedat ON library.entry (userid, updatedat DESC);

COMMENT ON TABLE  library.entry IS 'Personal comic shelf entry. Created when a user follows/bookmarks a comic. Tracks reading status, personal score, and last read position.';
COMMENT ON COLUMN library.entry.readingstatus IS 'Shelf category. Allowed values (Go): reading | completed | on_hold | dropped | plan_to_read.';
COMMENT ON COLUMN library.entry.score IS 'Private personal score 1–10 (Go-validated). Distinct from public social.comicrating.';
COMMENT ON COLUMN library.entry.hasnew IS 'TRUE = unread chapters available since last visit. Updated by background job.';
COMMENT ON COLUMN library.entry.lastreadchapterid IS 'Denormalized from readingprogress for fast "Continue Reading" button (avoids extra join).';
COMMENT ON COLUMN library.entry.notes IS 'Private freeform notes, visible only to the owner.';

-- ============================================================
-- library.customlist  — User-curated named reading lists
-- ============================================================
CREATE TABLE library.customlist (
    id          TEXT        NOT NULL,
    userid      TEXT        NOT NULL,
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    visibility  VARCHAR(10) NOT NULL DEFAULT 'private',
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deletedat   TIMESTAMPTZ,

    CONSTRAINT customlist_pkey    PRIMARY KEY (id),
    CONSTRAINT customlist_user_fk FOREIGN KEY (userid) REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('library', 'customlist');

CREATE INDEX idx_library_customlist_userid ON library.customlist (userid) WHERE deletedat IS NULL;

COMMENT ON TABLE  library.customlist IS 'User-curated reading lists — similar to MDLists on MangaDex. Can be shared publicly or kept private.';
COMMENT ON COLUMN library.customlist.id IS 'UUIDv7. Used in shareable URLs: /list/{id}.';
COMMENT ON COLUMN library.customlist.visibility IS 'Access level. Allowed values (Go): public | private | unlisted.';

-- ============================================================
-- library.customlistitem  — Comics inside a custom list
-- ============================================================
CREATE TABLE library.customlistitem (
    listid      TEXT        NOT NULL,
    comicid     TEXT        NOT NULL,
    sortorder   INTEGER     NOT NULL DEFAULT 0,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT customlistitem_pkey     PRIMARY KEY (listid, comicid),
    CONSTRAINT customlistitem_list_fk  FOREIGN KEY (listid)  REFERENCES library.customlist (id) ON DELETE CASCADE,
    CONSTRAINT customlistitem_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic         (id) ON DELETE CASCADE
);

CREATE INDEX idx_library_customlistitem_comicid ON library.customlistitem (comicid);

COMMENT ON TABLE  library.customlistitem IS 'M:N junction between customlist and comic. sortorder controls display sequence.';
COMMENT ON COLUMN library.customlistitem.sortorder IS 'Manual ordering within the list. Lower = shown first. Default 0.';

-- ============================================================
-- library.readingprogress  — Last chapter/page per user per comic
-- ============================================================
CREATE TABLE library.readingprogress (
    id          BIGSERIAL   NOT NULL,
    userid      TEXT        NOT NULL,
    comicid     TEXT        NOT NULL,
    chapterid   TEXT        NOT NULL,
    pagenumber  SMALLINT    NOT NULL DEFAULT 1,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT readingprogress_pkey     PRIMARY KEY (id),
    CONSTRAINT readingprogress_uq       UNIQUE (userid, comicid),
    CONSTRAINT readingprogress_user_fk  FOREIGN KEY (userid)    REFERENCES users.account (id) ON DELETE CASCADE,
    CONSTRAINT readingprogress_comic_fk FOREIGN KEY (comicid)   REFERENCES core.comic   (id) ON DELETE CASCADE,
    CONSTRAINT readingprogress_chap_fk  FOREIGN KEY (chapterid) REFERENCES core.chapter (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('library', 'readingprogress');

CREATE INDEX idx_library_readingprogress_userid ON library.readingprogress (userid);

COMMENT ON TABLE  library.readingprogress IS 'Tracks the furthest reading position per (user, comic). Upserted on every page turn. Drives the "Continue Reading" feature; denormalized into library.entry.lastreadchapterid.';
COMMENT ON COLUMN library.readingprogress.pagenumber IS 'Last page reached. Default 1 = chapter opened, no specific page tracked yet.';

-- ============================================================
-- library.chapterread  — Per-chapter read completion record
-- ============================================================
CREATE TABLE library.chapterread (
    id          BIGSERIAL   NOT NULL,
    userid      TEXT        NOT NULL,
    chapterid   TEXT        NOT NULL,
    readat      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chapterread_pkey    PRIMARY KEY (id),
    CONSTRAINT chapterread_uq      UNIQUE (userid, chapterid),
    CONSTRAINT chapterread_user_fk FOREIGN KEY (userid)    REFERENCES users.account (id) ON DELETE CASCADE,
    CONSTRAINT chapterread_chap_fk FOREIGN KEY (chapterid) REFERENCES core.chapter  (id) ON DELETE CASCADE
);

CREATE INDEX idx_library_chapterread_userid    ON library.chapterread (userid);
CREATE INDEX idx_library_chapterread_chapterid ON library.chapterread (chapterid);

COMMENT ON TABLE  library.chapterread IS 'Append-style record of chapters a user has fully read. Powers the checkmark indicator on chapter list rows. One row per (user, chapter); never updated, only inserted.';
COMMENT ON COLUMN library.chapterread.readat IS 'When the chapter was marked complete (last page reached or explicit "mark read").';

-- ============================================================
-- library.viewhistory  — Browsing history (recently viewed comics)
-- ============================================================
-- Different from readingprogress (which tracks CHAPTERS):
-- viewhistory tracks when a user VIEWED a comic detail page.
-- Powers the "Recently Viewed" / "History" section in user profile.
-- Capped at last 500 entries per user (eviction by Go background job).
-- ============================================================
CREATE TABLE library.viewhistory (
    id        BIGSERIAL   NOT NULL,
    userid    TEXT        NOT NULL,
    comicid   TEXT        NOT NULL,
    viewedat  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT viewhistory_pkey     PRIMARY KEY (id),
    CONSTRAINT viewhistory_user_fk  FOREIGN KEY (userid)  REFERENCES users.account (id) ON DELETE CASCADE,
    CONSTRAINT viewhistory_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic    (id) ON DELETE CASCADE
);

CREATE INDEX idx_library_viewhistory_userid  ON library.viewhistory (userid, viewedat DESC);
CREATE INDEX idx_library_viewhistory_comicid ON library.viewhistory (comicid);

COMMENT ON TABLE  library.viewhistory IS 'Browsing history: records when a user viewed a comic detail page.
     Powers the "Recently Viewed" section in user profile.
     Different from readingprogress (chapter-level) — this is comic-level.
     Capped at 500 entries per user by Go background cleanup job.';
COMMENT ON COLUMN library.viewhistory.viewedat IS 'When the user visited the comic detail page. Latest visit = most recent row.';

-- =============================================================================
-- END OF 30_LIBRARY
-- =============================================================================
