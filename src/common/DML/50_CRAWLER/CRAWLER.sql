-- =============================================================================
-- YOMIRA — 50_CRAWLER
-- Schema: crawler
-- @author  tai.buivan.jp@gmail.com
-- Tables: source · comicsource · job · log (partitioned)
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql · 10_USERS/USERS.sql · 20_CORE/CORE.sql
-- =============================================================================

-- ============================================================
-- crawler.source  — Registered external source websites
-- ============================================================
CREATE SEQUENCE crawler.source_id_seq START 100000 INCREMENT 1;

CREATE TABLE crawler.source (
    id               INTEGER     NOT NULL DEFAULT nextval('crawler.source_id_seq'),
    name             VARCHAR(100) NOT NULL,
    slug             VARCHAR(120) NOT NULL,
    baseurl          TEXT        NOT NULL,
    extensionid      VARCHAR(200),
    config           JSONB       NOT NULL DEFAULT '{}',
    isenabled        BOOLEAN     NOT NULL DEFAULT TRUE,
    lastsucceededat  TIMESTAMPTZ,
    lastfailedat     TIMESTAMPTZ,
    consecutivefails SMALLINT    NOT NULL DEFAULT 0,
    createdat        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT source_pkey    PRIMARY KEY (id),
    CONSTRAINT source_name_uq UNIQUE (name),
    CONSTRAINT source_slug_uq UNIQUE (slug)
);
SELECT attach_updatedat_trigger('crawler', 'source');

COMMENT ON TABLE  crawler.source IS 'Registered external comic websites the crawler engine can pull from. Each row corresponds to one Go extension plugin.';
COMMENT ON COLUMN crawler.source.extensionid IS 'Reverse-domain plugin ID loaded by the Go extension loader, e.g. "com.yomira.extension.nettruyen".';
COMMENT ON COLUMN crawler.source.config IS 'JSONB per-extension configuration: rate limits, HTTP headers, CSS selectors, proxy settings. Schema is extension-specific; validated by Go, not by DB.';
COMMENT ON COLUMN crawler.source.consecutivefails IS 'Counter of back-to-back failed jobs. Go auto-disables source when threshold exceeded.';
COMMENT ON COLUMN crawler.source.isenabled IS 'FALSE = source skipped by scheduler. Can be toggled by admin without deletion.';

-- ============================================================
-- crawler.comicsource  — Maps a comic to its URL(s) on source sites
-- ============================================================
CREATE TABLE crawler.comicsource (
    id           BIGSERIAL   NOT NULL,
    comicid      TEXT        NOT NULL,
    sourceid     INTEGER     NOT NULL,
    sourceid_ext TEXT        NOT NULL,
    sourceurl    TEXT        NOT NULL,
    isactive     BOOLEAN     NOT NULL DEFAULT TRUE,
    lastcrawlat  TIMESTAMPTZ,
    createdat    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT comicsource_pkey      PRIMARY KEY (id),
    CONSTRAINT comicsource_uq        UNIQUE (sourceid, sourceid_ext),
    CONSTRAINT comicsource_comic_fk  FOREIGN KEY (comicid)  REFERENCES core.comic     (id) ON DELETE CASCADE,
    CONSTRAINT comicsource_source_fk FOREIGN KEY (sourceid) REFERENCES crawler.source (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('crawler', 'comicsource');

CREATE INDEX idx_crawler_comicsource_comicid  ON crawler.comicsource (comicid);
CREATE INDEX idx_crawler_comicsource_sourceid ON crawler.comicsource (sourceid);

COMMENT ON TABLE  crawler.comicsource IS 'Links one comic to one or more external source URLs. A comic may exist on multiple sources (e.g. both NetTruyen and MangaDex).';
COMMENT ON COLUMN crawler.comicsource.sourceid_ext IS 'The source site''s own identifier for this comic — slug or internal numeric ID.';
COMMENT ON COLUMN crawler.comicsource.lastcrawlat IS 'Timestamp of the most recent successful crawl for this (comic, source) pair.';

-- ============================================================
-- crawler.job  — Lifecycle record for one crawl operation
-- ============================================================
CREATE TABLE crawler.job (
    id          TEXT        NOT NULL,
    sourceid    INTEGER     NOT NULL,
    comicid     TEXT,
    status      VARCHAR(20) NOT NULL DEFAULT 'queued',
    scheduledat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    startedat   TIMESTAMPTZ,
    finishedat  TIMESTAMPTZ,
    pagescount  INTEGER     NOT NULL DEFAULT 0,
    errorcount  INTEGER     NOT NULL DEFAULT 0,
    lasterror   TEXT,
    triggeredby TEXT,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT job_pkey       PRIMARY KEY (id),
    CONSTRAINT job_source_fk  FOREIGN KEY (sourceid)    REFERENCES crawler.source (id) ON DELETE CASCADE,
    CONSTRAINT job_comic_fk   FOREIGN KEY (comicid)     REFERENCES core.comic     (id) ON DELETE SET NULL,
    CONSTRAINT job_trigger_fk FOREIGN KEY (triggeredby) REFERENCES users.account  (id) ON DELETE SET NULL
);
SELECT attach_updatedat_trigger('crawler', 'job');

CREATE INDEX idx_crawler_job_status      ON crawler.job (status);
CREATE INDEX idx_crawler_job_sourceid    ON crawler.job (sourceid);
CREATE INDEX idx_crawler_job_scheduledat ON crawler.job (scheduledat);

COMMENT ON TABLE  crawler.job IS 'One row per crawl execution. NULL comicid = full-catalogue discovery crawl.';
COMMENT ON COLUMN crawler.job.id IS 'UUIDv7. Used as correlation ID in crawler.log entries.';
COMMENT ON COLUMN crawler.job.status IS 'Job lifecycle state. Allowed values (Go): queued | running | done | failed | cancelled.';
COMMENT ON COLUMN crawler.job.pagescount IS 'Total pages/chapters successfully fetched during this job run.';
COMMENT ON COLUMN crawler.job.triggeredby IS 'NULL = automated scheduler. Non-NULL = admin user ID who triggered manually.';
COMMENT ON COLUMN crawler.job.lasterror IS 'Last error message from the run. Kept for quick debugging without querying crawler.log.';

-- ============================================================
-- crawler.log  — Append-only event log (partitioned by month)
-- ============================================================
-- Partition strategy: one partition per calendar month.
-- Monthly partitions must be created in advance (or via pg_partman).
-- Old partitions (> 6 months) can be detached and archived.
-- DO NOT issue UPDATE or DELETE on this table.
-- ============================================================
CREATE TABLE crawler.log (
    id          TEXT        NOT NULL,
    jobid       TEXT        NOT NULL,
    level       VARCHAR(10) NOT NULL DEFAULT 'info',
    message     TEXT        NOT NULL,
    meta        JSONB,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT log_pk     PRIMARY KEY (id, createdat),
    CONSTRAINT log_job_fk FOREIGN KEY (jobid) REFERENCES crawler.job (id) ON DELETE CASCADE
) PARTITION BY RANGE (createdat);

-- Default catch-all + monthly partitions
CREATE TABLE crawler.log_default PARTITION OF crawler.log DEFAULT;
CREATE TABLE crawler.log_2026_01 PARTITION OF crawler.log FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE crawler.log_2026_02 PARTITION OF crawler.log FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE crawler.log_2026_03 PARTITION OF crawler.log FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE crawler.log_2026_04 PARTITION OF crawler.log FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE crawler.log_2026_05 PARTITION OF crawler.log FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE crawler.log_2026_06 PARTITION OF crawler.log FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE INDEX idx_crawler_log_jobid     ON crawler.log (jobid, createdat);
CREATE INDEX idx_crawler_log_level     ON crawler.log (level);
CREATE INDEX idx_crawler_log_createdat ON crawler.log (createdat);

COMMENT ON TABLE  crawler.log IS 'Append-only structured log for crawl job events. Partitioned by month for efficient time-based pruning. Never UPDATE or DELETE individual rows — drop old partitions instead.';
COMMENT ON COLUMN crawler.log.level IS 'Log severity. Allowed values (Go): debug | info | warn | error.';
COMMENT ON COLUMN crawler.log.meta IS 'Arbitrary structured context as JSONB: URL, HTTP status, retry count, page number, etc.';

-- =============================================================================
-- END OF 50_CRAWLER
-- =============================================================================
