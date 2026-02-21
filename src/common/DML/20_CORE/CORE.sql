-- =============================================================================
-- YOMIRA — 20_CORE
-- Schema: core
-- @author  tai.buivan.jp@gmail.com
-- Tables (19):
--   language · author · artist · taggroup · tag
--   scanlationgroup · scanlationgroupmember · scanlationgroupfollow
--   comic · comictitle · comicrelation · comicauthor · comicartist · comictag
--   comiccover · comicart · chapter · page · mediafile
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql · 10_USERS/USERS.sql
-- =============================================================================

-- ============================================================
-- core.language  — ISO / BCP-47 language reference (static)
-- ============================================================
CREATE SEQUENCE core.language_id_seq START 100000 INCREMENT 1;
CREATE TABLE core.language (
    id         INTEGER      NOT NULL DEFAULT nextval('core.language_id_seq'),
    code       VARCHAR(10)  NOT NULL,
    name       VARCHAR(80)  NOT NULL,
    nativename VARCHAR(80),
    CONSTRAINT language_pkey    PRIMARY KEY (id),
    CONSTRAINT language_code_uq UNIQUE (code)
);
COMMENT ON TABLE  core.language IS 'BCP-47 language reference. Static data seeded at init; rarely changes.';
COMMENT ON COLUMN core.language.code IS 'BCP-47 code, e.g. "en", "vi", "ja", "zh-hans". Primary lookup key.';
COMMENT ON COLUMN core.language.nativename IS 'Self-name of the language in its own script, e.g. "Tiếng Việt", "日本語".';

-- ============================================================
-- core.author  — Story writer
-- ============================================================
CREATE SEQUENCE core.author_id_seq START 100000 INCREMENT 1;
CREATE TABLE core.author (
    id        INTEGER      NOT NULL DEFAULT nextval('core.author_id_seq'),
    name      VARCHAR(200) NOT NULL,
    namealt   TEXT[],
    bio       TEXT,
    imageurl  TEXT,
    createdat TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deletedat TIMESTAMPTZ,
    CONSTRAINT author_pkey PRIMARY KEY (id)
);
SELECT attach_updatedat_trigger('core', 'author');
CREATE INDEX idx_core_author_name ON core.author USING gin (name gin_trgm_ops);
COMMENT ON TABLE  core.author IS 'Scenario writer. One author may be linked to many comics via core.comicauthor.';
COMMENT ON COLUMN core.author.namealt IS 'Array of alternative names: romanized, original-script, pen names.';

-- ============================================================
-- core.artist  — Illustrator / art team
-- ============================================================
CREATE SEQUENCE core.artist_id_seq START 100000 INCREMENT 1;
CREATE TABLE core.artist (
    id        INTEGER      NOT NULL DEFAULT nextval('core.artist_id_seq'),
    name      VARCHAR(200) NOT NULL,
    namealt   TEXT[],
    bio       TEXT,
    imageurl  TEXT,
    createdat TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deletedat TIMESTAMPTZ,
    CONSTRAINT artist_pkey PRIMARY KEY (id)
);
SELECT attach_updatedat_trigger('core', 'artist');
CREATE INDEX idx_core_artist_name ON core.artist USING gin (name gin_trgm_ops);
COMMENT ON TABLE  core.artist IS 'Illustrator. Separate from author because many manga split the writer/artist roles.';
COMMENT ON COLUMN core.artist.namealt IS 'Alternative names: romanized, original-script, pen names.';

-- ============================================================
-- core.taggroup  — Tag category (MangaDex 4-tier)
-- ============================================================
CREATE SEQUENCE core.taggroup_id_seq START 100000 INCREMENT 1;
CREATE TABLE core.taggroup (
    id        INTEGER     NOT NULL DEFAULT nextval('core.taggroup_id_seq'),
    name      VARCHAR(50) NOT NULL,
    slug      VARCHAR(60) NOT NULL,
    sortorder SMALLINT    NOT NULL DEFAULT 0,
    CONSTRAINT taggroup_pkey    PRIMARY KEY (id),
    CONSTRAINT taggroup_slug_uq UNIQUE (slug)
);
COMMENT ON TABLE  core.taggroup IS 'Groups tags by purpose. Seed data: Genre / Theme / Format / Demographic. Mirrors MangaDex 4-tier tag system.';
COMMENT ON COLUMN core.taggroup.sortorder IS 'Controls display order in filter UI. Lower = shown first.';

-- ============================================================
-- core.tag  — Unified classification tag
-- ============================================================
-- Replaces separate "genre" and "tag" tables from v1.
-- A single comictag junction covers all 4 groups.
-- ============================================================
CREATE SEQUENCE core.tag_id_seq START 100000 INCREMENT 1;
CREATE TABLE core.tag (
    id          INTEGER      NOT NULL DEFAULT nextval('core.tag_id_seq'),
    groupid     INTEGER      NOT NULL,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(120) NOT NULL,
    description TEXT,
    createdat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT tag_pkey     PRIMARY KEY (id),
    CONSTRAINT tag_slug_uq  UNIQUE (slug),
    CONSTRAINT tag_group_fk FOREIGN KEY (groupid) REFERENCES core.taggroup (id)
);
SELECT attach_updatedat_trigger('core', 'tag');
CREATE INDEX idx_core_tag_groupid ON core.tag (groupid);
COMMENT ON TABLE  core.tag IS 'Unified tag. Replaces separate genre+tag tables from v1. Each tag belongs to a taggroup (Genre, Theme, Format, or Demographic).';
COMMENT ON COLUMN core.tag.groupid IS 'Foreign key to taggroup. Determines which filter section this tag appears in.';
COMMENT ON COLUMN core.tag.slug IS 'URL-safe unique identifier, e.g. "slice-of-life", "long-strip".';

-- ============================================================
-- core.scanlationgroup  — Translation / scanlation teams
-- ============================================================
CREATE TABLE core.scanlationgroup (
    id                  TEXT         NOT NULL,
    name                VARCHAR(200) NOT NULL,
    slug                VARCHAR(220) NOT NULL,
    description         TEXT,
    website             TEXT,
    ircserver           TEXT,
    irchannel           TEXT,
    discord             TEXT,
    email               TEXT,
    patreon             TEXT,
    twitter             TEXT,
    youtube             TEXT,
    mangaupdates        TEXT,
    isofficialpublisher BOOLEAN      NOT NULL DEFAULT FALSE,
    isactive            BOOLEAN      NOT NULL DEFAULT TRUE,
    isfocused           BOOLEAN      NOT NULL DEFAULT FALSE,
    verifiedat          TIMESTAMPTZ,
    verifiedby          TEXT,
    createdat           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deletedat           TIMESTAMPTZ,
    CONSTRAINT scanlationgroup_pkey    PRIMARY KEY (id),
    CONSTRAINT scanlationgroup_slug_uq UNIQUE (slug)
);
SELECT attach_updatedat_trigger('core', 'scanlationgroup');
COMMENT ON TABLE  core.scanlationgroup IS 'Translation or scanlation team. May be a community group or an official publisher.';
COMMENT ON COLUMN core.scanlationgroup.isofficialpublisher IS 'TRUE = verified official publisher (e.g. MangaPlus/Shonen Jump). Shows badge on chapters.';
COMMENT ON COLUMN core.scanlationgroup.isfocused IS 'TRUE = group is currently active / accepting new translation projects.';
COMMENT ON COLUMN core.scanlationgroup.verifiedby IS 'users.account.id of the admin who granted official-publisher status.';
COMMENT ON COLUMN core.scanlationgroup.mangaupdates IS 'MangaUpdates group ID for cross-referencing the external scanlation database.';

-- ============================================================
-- core.scanlationgroupmember  — Group membership and roles
-- ============================================================
CREATE TABLE core.scanlationgroupmember (
    groupid   TEXT        NOT NULL,
    userid    TEXT        NOT NULL,
    role      VARCHAR(30) NOT NULL DEFAULT 'member',
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scanlationgroupmember_pkey     PRIMARY KEY (groupid, userid),
    CONSTRAINT scanlationgroupmember_group_fk FOREIGN KEY (groupid) REFERENCES core.scanlationgroup (id) ON DELETE CASCADE,
    CONSTRAINT scanlationgroupmember_user_fk  FOREIGN KEY (userid)  REFERENCES users.account        (id) ON DELETE CASCADE
);
CREATE INDEX idx_core_scanlationgroupmember_userid ON core.scanlationgroupmember (userid);
COMMENT ON TABLE  core.scanlationgroupmember IS 'Membership of a user in a scanlation group. Manages internal group roles.';
COMMENT ON COLUMN core.scanlationgroupmember.role IS 'Member role. Allowed values (Go): leader | moderator | member.';

-- ============================================================
-- core.scanlationgroupfollow  — Users following a group
-- ============================================================
CREATE TABLE core.scanlationgroupfollow (
    userid    TEXT        NOT NULL,
    groupid   TEXT        NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scanlationgroupfollow_pkey     PRIMARY KEY (userid, groupid),
    CONSTRAINT scanlationgroupfollow_user_fk  FOREIGN KEY (userid)  REFERENCES users.account        (id) ON DELETE CASCADE,
    CONSTRAINT scanlationgroupfollow_group_fk FOREIGN KEY (groupid) REFERENCES core.scanlationgroup (id) ON DELETE CASCADE
);
CREATE INDEX idx_core_scanlationgroupfollow_groupid ON core.scanlationgroupfollow (groupid);
COMMENT ON TABLE  core.scanlationgroupfollow IS 'Users who follow a scanlation group. New chapters from the group appear in the follower''s activity Feed.';

-- ============================================================
-- core.comic  — Manga / manhwa / webtoon series
-- ============================================================
CREATE TABLE core.comic (
    id              TEXT          NOT NULL,
    title           VARCHAR(500)  NOT NULL,
    titlealt        TEXT[],
    slug            VARCHAR(520)  NOT NULL,
    synopsis        TEXT,
    coverurl        TEXT,
    bannerurl       TEXT,
    status          VARCHAR(20)   NOT NULL DEFAULT 'unknown',
    contentrating   VARCHAR(20)   NOT NULL DEFAULT 'safe',
    demographic     VARCHAR(20),
    defaultreadmode VARCHAR(20)   NOT NULL DEFAULT 'ltr',
    originlanguage  VARCHAR(10),
    year            SMALLINT,
    links           JSONB         NOT NULL DEFAULT '{}',
    viewcount       BIGINT        NOT NULL DEFAULT 0,
    followcount     BIGINT        NOT NULL DEFAULT 0,
    chaptercount    INTEGER       NOT NULL DEFAULT 0,
    ratingavg       NUMERIC(4,2)  NOT NULL DEFAULT 0,
    ratingbayesian  NUMERIC(4,2)  NOT NULL DEFAULT 0,
    ratingcount     INTEGER       NOT NULL DEFAULT 0,
    islocked        BOOLEAN       NOT NULL DEFAULT FALSE,
    searchvector    TSVECTOR,
    createdat       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updatedat       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deletedat       TIMESTAMPTZ,
    CONSTRAINT comic_pkey    PRIMARY KEY (id),
    CONSTRAINT comic_slug_uq UNIQUE (slug)
);
SELECT attach_updatedat_trigger('core', 'comic');

CREATE INDEX idx_core_comic_status       ON core.comic (status)          WHERE deletedat IS NULL;
CREATE INDEX idx_core_comic_rating       ON core.comic (contentrating)   WHERE deletedat IS NULL;
CREATE INDEX idx_core_comic_demographic  ON core.comic (demographic)     WHERE deletedat IS NULL AND demographic IS NOT NULL;
CREATE INDEX idx_core_comic_originlang   ON core.comic (originlanguage)  WHERE deletedat IS NULL AND originlanguage IS NOT NULL;
CREATE INDEX idx_core_comic_createdat    ON core.comic (createdat)       WHERE deletedat IS NULL;
CREATE INDEX idx_core_comic_viewcount    ON core.comic (viewcount DESC)  WHERE deletedat IS NULL;
CREATE INDEX idx_core_comic_followcount  ON core.comic (followcount DESC) WHERE deletedat IS NULL;
CREATE INDEX idx_core_comic_ratingavg    ON core.comic (ratingavg DESC)  WHERE deletedat IS NULL;
CREATE INDEX idx_core_comic_year         ON core.comic (year)            WHERE deletedat IS NULL AND year IS NOT NULL;
CREATE INDEX idx_core_comic_searchvector ON core.comic USING gin (searchvector);
CREATE INDEX idx_core_comic_title_trgm   ON core.comic USING gin (title gin_trgm_ops);

-- Full-text search trigger: title(A) + titlealt(A) + synopsis(C)
CREATE OR REPLACE FUNCTION core.comic_searchvector_update()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.searchvector :=
        setweight(to_tsvector('simple', unaccent(COALESCE(NEW.title, ''))), 'A') ||
        setweight(to_tsvector('simple', unaccent(array_to_string(COALESCE(NEW.titlealt, '{}'), ' '))), 'A') ||
        setweight(to_tsvector('simple', unaccent(COALESCE(NEW.synopsis, ''))), 'C');
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_comic_searchvector
    BEFORE INSERT OR UPDATE OF title, titlealt, synopsis
    ON core.comic
    FOR EACH ROW EXECUTE FUNCTION core.comic_searchvector_update();

COMMENT ON TABLE  core.comic IS 'Core content entity: a manga/manhwa/webtoon series. All metadata, counters, and search index for a title live here.';
COMMENT ON COLUMN core.comic.id IS 'UUIDv7. Used in all public URLs: /title/{id}.';
COMMENT ON COLUMN core.comic.titlealt IS 'Synonyms and alternate-script titles. Included in searchvector (weight A).';
COMMENT ON COLUMN core.comic.status IS 'Publication state (Go): ongoing | completed | hiatus | cancelled | unknown.';
COMMENT ON COLUMN core.comic.contentrating IS 'Maturity level (Go): safe | suggestive | explicit. Controls NSFW filter.';
COMMENT ON COLUMN core.comic.demographic IS 'Target readership (Go): shounen | shoujo | seinen | josei | none.';
COMMENT ON COLUMN core.comic.defaultreadmode IS 'Suggested read direction (Go): ltr | rtl | vertical | webtoon.';
COMMENT ON COLUMN core.comic.originlanguage IS 'BCP-47 code of original publication: "ja"=manga, "ko"=manhwa, "zh"=manhua.';
COMMENT ON COLUMN core.comic.links IS 'External links JSONB map: {"mal":"12345","anilist":"67890","official":"https://..."}.';
COMMENT ON COLUMN core.comic.viewcount IS 'Denormalized. Buffered in Redis INCR; flushed to DB every ~60s by background job.';
COMMENT ON COLUMN core.comic.followcount IS 'Denormalized. Updated directly on follow/unfollow (low-contention direct UPDATE).';
COMMENT ON COLUMN core.comic.ratingavg IS 'Running arithmetic average. Updated via: (ratingavg*ratingcount + score)/(ratingcount+1).';
COMMENT ON COLUMN core.comic.ratingbayesian IS 'Bayesian-adjusted average: (C*m + Σscores)/(C+n). Recomputed by background job.';
COMMENT ON COLUMN core.comic.islocked IS 'Admin flag. TRUE = community metadata edits blocked; only admin can modify.';
COMMENT ON COLUMN core.comic.searchvector IS 'tsvector maintained by trigger on INSERT/UPDATE of title, titlealt, synopsis.';

-- ============================================================
-- core.comictitle  — Per-language title & description
-- ============================================================
-- MangaDex stores titles as a language map: {"en": "...", "ja": "..."}.
-- This table normalises that into rows for easy querying and indexing.
-- The primary/default title is still stored in core.comic.title (Go selects
-- based on user language preference → fallback to 'en' → fallback to comic.title).
-- ============================================================
CREATE TABLE core.comictitle (
    comicid     TEXT         NOT NULL,
    languageid  INTEGER      NOT NULL,
    title       VARCHAR(500) NOT NULL,
    description TEXT,
    createdat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT comictitle_pkey     PRIMARY KEY (comicid, languageid),
    CONSTRAINT comictitle_comic_fk FOREIGN KEY (comicid)    REFERENCES core.comic    (id) ON DELETE CASCADE,
    CONSTRAINT comictitle_lang_fk  FOREIGN KEY (languageid) REFERENCES core.language (id)
);
SELECT attach_updatedat_trigger('core', 'comictitle');
CREATE INDEX idx_core_comictitle_comicid ON core.comictitle (comicid);
CREATE INDEX idx_core_comictitle_title   ON core.comictitle USING gin (title gin_trgm_ops);

COMMENT ON TABLE  core.comictitle IS 'Per-language titles and descriptions for a comic. Mirrors MangaDex multilingual title map: {"en": "One Piece", "ja": "ワンピース"}. Primary title fallback order (Go): user language → en → comic.title.';
COMMENT ON COLUMN core.comictitle.languageid IS 'Which language this title/description is written in.';
COMMENT ON COLUMN core.comictitle.title IS 'Title in the specified language. Included in trigram search.';
COMMENT ON COLUMN core.comictitle.description IS 'Synopsis/description in the specified language. NULL = use core.comic.synopsis.';

-- ============================================================
-- core.comicrelation  — Directed relationship between 2 comics
-- ============================================================
CREATE TABLE core.comicrelation (
    comicid        TEXT        NOT NULL,
    relatedcomicid TEXT        NOT NULL,
    relationtype   VARCHAR(30) NOT NULL,
    createdat      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comicrelation_pkey       PRIMARY KEY (comicid, relatedcomicid, relationtype),
    CONSTRAINT comicrelation_comic_fk   FOREIGN KEY (comicid)        REFERENCES core.comic (id) ON DELETE CASCADE,
    CONSTRAINT comicrelation_related_fk FOREIGN KEY (relatedcomicid) REFERENCES core.comic (id) ON DELETE CASCADE
);
COMMENT ON TABLE  core.comicrelation IS 'Directed relation between two comics. Self-reference prevented by Go service.';
COMMENT ON COLUMN core.comicrelation.relationtype IS 'Allowed values (Go): sequel | prequel | main_story | side_story | spin_off | adaptation | alternate | same_franchise | colored | preserialization.';

-- ============================================================
-- core.comicauthor  — M:N  comic ↔ author
-- ============================================================
CREATE TABLE core.comicauthor (
    comicid   TEXT        NOT NULL,
    authorid  INTEGER     NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comicauthor_pkey      PRIMARY KEY (comicid, authorid),
    CONSTRAINT comicauthor_comic_fk  FOREIGN KEY (comicid)  REFERENCES core.comic  (id) ON DELETE CASCADE,
    CONSTRAINT comicauthor_author_fk FOREIGN KEY (authorid) REFERENCES core.author (id) ON DELETE CASCADE
);
COMMENT ON TABLE core.comicauthor IS 'M:N junction: links a comic to its story writers.';

-- ============================================================
-- core.comicartist  — M:N  comic ↔ artist
-- ============================================================
CREATE TABLE core.comicartist (
    comicid   TEXT        NOT NULL,
    artistid  INTEGER     NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comicartist_pkey      PRIMARY KEY (comicid, artistid),
    CONSTRAINT comicartist_comic_fk  FOREIGN KEY (comicid)  REFERENCES core.comic  (id) ON DELETE CASCADE,
    CONSTRAINT comicartist_artist_fk FOREIGN KEY (artistid) REFERENCES core.artist (id) ON DELETE CASCADE
);
COMMENT ON TABLE core.comicartist IS 'M:N junction: links a comic to its illustrators.';

-- ============================================================
-- core.comictag  — M:N  comic ↔ tag  (all groups)
-- ============================================================
CREATE TABLE core.comictag (
    comicid   TEXT        NOT NULL,
    tagid     INTEGER     NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comictag_pkey     PRIMARY KEY (comicid, tagid),
    CONSTRAINT comictag_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic (id) ON DELETE CASCADE,
    CONSTRAINT comictag_tag_fk   FOREIGN KEY (tagid)   REFERENCES core.tag   (id) ON DELETE CASCADE
);
CREATE INDEX idx_core_comictag_tagid ON core.comictag (tagid);
COMMENT ON TABLE core.comictag IS 'M:N junction: links a comic to tags of any group (Genre, Theme, Format, Demographic). Single table replaces the former separate comicgenre + comictag approach.';

-- ============================================================
-- core.comiccover  — Volume cover images
-- ============================================================
CREATE TABLE core.comiccover (
    id          TEXT        NOT NULL,
    comicid     TEXT        NOT NULL,
    volume      SMALLINT,
    imageurl    TEXT        NOT NULL,
    description TEXT,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comiccover_pkey     PRIMARY KEY (id),
    CONSTRAINT comiccover_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('core', 'comiccover');
CREATE INDEX idx_core_comiccover_comicid ON core.comiccover (comicid);
COMMENT ON TABLE  core.comiccover IS 'Per-volume cover images. NULL volume = default series cover (no specific volume).';
COMMENT ON COLUMN core.comiccover.volume IS 'Volume number. NULL = default cover image used when no volume is selected.';

-- ============================================================
-- core.comicart  — Art gallery (Art tab)
-- ============================================================
CREATE TABLE core.comicart (
    id          TEXT        NOT NULL,
    comicid     TEXT        NOT NULL,
    uploaderid  TEXT,
    arttype     VARCHAR(30) NOT NULL DEFAULT 'cover',
    volume      SMALLINT,
    imageurl    TEXT        NOT NULL,
    width       SMALLINT,
    height      SMALLINT,
    description TEXT,
    isapproved  BOOLEAN     NOT NULL DEFAULT TRUE,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deletedat   TIMESTAMPTZ,
    CONSTRAINT comicart_pkey     PRIMARY KEY (id),
    CONSTRAINT comicart_comic_fk FOREIGN KEY (comicid)    REFERENCES core.comic    (id) ON DELETE CASCADE,
    CONSTRAINT comicart_user_fk  FOREIGN KEY (uploaderid) REFERENCES users.account (id) ON DELETE SET NULL
);
SELECT attach_updatedat_trigger('core', 'comicart');
CREATE INDEX idx_core_comicart_comicid ON core.comicart (comicid, arttype) WHERE deletedat IS NULL;
COMMENT ON TABLE  core.comicart IS 'Art gallery for a comic, shown in the "Art" tab. Includes volume covers, promotional art, and community fanart.';
COMMENT ON COLUMN core.comicart.arttype IS 'Image category (Go): cover | banner | fanart | promotional | chapter_cover.';
COMMENT ON COLUMN core.comicart.isapproved IS 'Moderation state. FALSE = pending review (fanart community submissions).';

-- ============================================================
-- core.chapter  — One translated chapter
-- ============================================================
-- Multi-language: same chapternumber may appear N times, one per language/group.
-- ============================================================
CREATE TABLE core.chapter (
    id                TEXT          NOT NULL,
    comicid           TEXT          NOT NULL,
    languageid        INTEGER,
    scanlationgroupid TEXT,
    uploaderid        TEXT,
    volume            SMALLINT,
    chapternumber     NUMERIC(8,2)  NOT NULL,
    title             VARCHAR(500),
    syncstate         VARCHAR(20)   NOT NULL DEFAULT 'pending',
    sourceurl         TEXT,
    externalurl       TEXT,
    isofficial        BOOLEAN       NOT NULL DEFAULT FALSE,
    islocked          BOOLEAN       NOT NULL DEFAULT FALSE,
    pagecount         INTEGER       NOT NULL DEFAULT 0,
    viewcount         BIGINT        NOT NULL DEFAULT 0,
    publishedat       TIMESTAMPTZ,
    createdat         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updatedat         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deletedat         TIMESTAMPTZ,
    CONSTRAINT chapter_pkey          PRIMARY KEY (id),
    CONSTRAINT chapter_comic_fk      FOREIGN KEY (comicid)          REFERENCES core.comic           (id) ON DELETE CASCADE,
    CONSTRAINT chapter_language_fk   FOREIGN KEY (languageid)        REFERENCES core.language        (id) ON DELETE SET NULL,
    CONSTRAINT chapter_scanlation_fk FOREIGN KEY (scanlationgroupid) REFERENCES core.scanlationgroup (id) ON DELETE SET NULL,
    CONSTRAINT chapter_uploader_fk   FOREIGN KEY (uploaderid)        REFERENCES users.account        (id) ON DELETE SET NULL
);
SELECT attach_updatedat_trigger('core', 'chapter');

CREATE INDEX idx_core_chapter_comicid     ON core.chapter (comicid, chapternumber) WHERE deletedat IS NULL;
CREATE INDEX idx_core_chapter_language    ON core.chapter (comicid, languageid)    WHERE deletedat IS NULL;
CREATE INDEX idx_core_chapter_syncstate   ON core.chapter (syncstate)              WHERE deletedat IS NULL;
CREATE INDEX idx_core_chapter_publishedat ON core.chapter (publishedat DESC)       WHERE deletedat IS NULL;
CREATE INDEX idx_core_chapter_viewcount   ON core.chapter (viewcount DESC)         WHERE deletedat IS NULL;
CREATE INDEX idx_core_chapter_group       ON core.chapter (scanlationgroupid, comicid, chapternumber) WHERE deletedat IS NULL;

COMMENT ON TABLE  core.chapter IS 'A single translated chapter. One comic can have multiple rows for the same chapternumber if different groups translated it into different languages.';
COMMENT ON COLUMN core.chapter.chapternumber IS 'NUMERIC(8,2) supports fractional chapters: 12.5 = interlude between 12 and 13.';
COMMENT ON COLUMN core.chapter.syncstate IS 'Crawler pipeline state (Go): pending | processing | synced | failed | missing.';
COMMENT ON COLUMN core.chapter.externalurl IS 'Direct deep-link to the original platform reader, e.g. MangaPlus chapter URL.';
COMMENT ON COLUMN core.chapter.isofficial IS 'TRUE = uploaded by a verified official publisher.';
COMMENT ON COLUMN core.chapter.islocked IS 'TRUE = no new community scanlations accepted (official version exists).';
COMMENT ON COLUMN core.chapter.publishedat IS 'Actual publication time (may be backdated for imported chapters; differs from createdat).';

-- ============================================================
-- core.page  — Individual page image within a chapter
-- ============================================================
CREATE TABLE core.page (
    id            TEXT         NOT NULL,
    chapterid     TEXT         NOT NULL,
    pagenumber    SMALLINT     NOT NULL,
    imageurl      TEXT         NOT NULL,
    imageurlhd    TEXT,
    format        VARCHAR(10)  NOT NULL DEFAULT 'webp',
    width         SMALLINT,
    height        SMALLINT,
    filesizebytes INTEGER,
    createdat     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT page_pkey       PRIMARY KEY (id),
    CONSTRAINT page_chapter_fk FOREIGN KEY (chapterid) REFERENCES core.chapter (id) ON DELETE CASCADE,
    CONSTRAINT page_order_uq   UNIQUE (chapterid, pagenumber)
);
SELECT attach_updatedat_trigger('core', 'page');
CREATE INDEX idx_core_page_chapterid ON core.page (chapterid, pagenumber);
COMMENT ON TABLE  core.page IS 'Individual page within a chapter. Ordered by pagenumber.';
COMMENT ON COLUMN core.page.imageurl IS 'Primary serving URL. Typically WebP after the image processing pipeline.';
COMMENT ON COLUMN core.page.imageurlhd IS 'High-res original URL. Served when user enables HD mode toggle in reader.';
COMMENT ON COLUMN core.page.format IS 'Image format (Go): jpeg | png | webp | gif | avif.';

-- ============================================================
-- core.mediafile  — Object storage file registry
-- ============================================================
CREATE TABLE core.mediafile (
    id              TEXT         NOT NULL,
    storagebucket   VARCHAR(100) NOT NULL,
    storagekey      TEXT         NOT NULL,
    publicurl       TEXT         NOT NULL,
    mimetype        VARCHAR(100) NOT NULL,
    format          VARCHAR(10),
    filesizebytes   INTEGER,
    width           SMALLINT,
    height          SMALLINT,
    sha256          CHAR(64),
    createdat       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT mediafile_pkey   PRIMARY KEY (id),
    CONSTRAINT mediafile_key_uq UNIQUE (storagebucket, storagekey),
    CONSTRAINT mediafile_sha_uq UNIQUE (sha256)
);
SELECT attach_updatedat_trigger('core', 'mediafile');
CREATE INDEX idx_core_mediafile_sha256    ON core.mediafile (sha256)    WHERE sha256 IS NOT NULL;
CREATE INDEX idx_core_mediafile_createdat ON core.mediafile (createdat);
COMMENT ON TABLE  core.mediafile IS 'Registry of all files in object storage (S3 / Cloudflare R2 / MinIO / local). sha256 deduplication prevents re-uploading identical images.';
COMMENT ON COLUMN core.mediafile.storagekey IS 'Object path within the bucket, e.g. "covers/comic/{uuid}/v1.webp".';
COMMENT ON COLUMN core.mediafile.sha256 IS 'SHA-256 hex digest of raw file bytes. Unique constraint prevents duplicates.';

-- =============================================================================
-- END OF 20_CORE
-- =============================================================================
