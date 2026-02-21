-- =============================================================================
-- YOMIRA — 60_ANALYTICS
-- Schema: analytics
-- @author  tai.buivan.jp@gmail.com
-- Tables: pageview · chaptersession  (both partitioned by month)
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql · 10_USERS/USERS.sql · 20_CORE/CORE.sql
--
-- NOTE: These tables are write-heavy and intentionally minimal.
-- Future plan: extract to ClickHouse / TimescaleDB when volume exceeds
-- ~100M rows/month. Until then, PostgreSQL range partitioning is sufficient.
-- =============================================================================

-- ============================================================
-- analytics.pageview  — Raw page view events
-- ============================================================
-- Partition by month. Anonymize ip/useragent after 90 days via background job.
-- ============================================================
CREATE TABLE analytics.pageview (
    id          BIGSERIAL   NOT NULL,
    entitytype  VARCHAR(20) NOT NULL,
    entityid    TEXT        NOT NULL,
    userid      TEXT,
    ipaddress   INET,
    useragent   TEXT,
    referrer    TEXT,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pageview_pk PRIMARY KEY (id, createdat)
) PARTITION BY RANGE (createdat);

CREATE TABLE analytics.pageview_default PARTITION OF analytics.pageview DEFAULT;
CREATE TABLE analytics.pageview_2026_02 PARTITION OF analytics.pageview FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE analytics.pageview_2026_03 PARTITION OF analytics.pageview FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE INDEX idx_analytics_pageview_entity    ON analytics.pageview (entitytype, entityid, createdat);
CREATE INDEX idx_analytics_pageview_createdat ON analytics.pageview (createdat);

COMMENT ON TABLE  analytics.pageview IS 'Raw page view events for comics and chapters. Write-heavy; partitioned by month for efficient pruning. Anonymize ip/useragent after 90 days per privacy policy. Scale path: migrate to ClickHouse when monthly volume > 100M rows.';
COMMENT ON COLUMN analytics.pageview.entitytype IS 'What was viewed. Allowed values (Go): comic | chapter.';
COMMENT ON COLUMN analytics.pageview.entityid IS 'UUID or ID of the viewed entity. Joined with core.comic or core.chapter on demand.';
COMMENT ON COLUMN analytics.pageview.userid IS 'NULL = anonymous visitor. Non-NULL = authenticated user UUID.';
COMMENT ON COLUMN analytics.pageview.referrer IS 'HTTP Referer header, truncated to 500 chars by Go before insert.';

-- ============================================================
-- analytics.chaptersession  — Granular reading session tracking
-- ============================================================
-- Captures open/close timestamps per chapter-read attempt.
-- Used for "time spent reading" metrics and completion-rate analysis.
-- Partitioned by startedat (= session open time).
-- ============================================================
CREATE TABLE analytics.chaptersession (
    id          BIGSERIAL   NOT NULL,
    chapterid   TEXT        NOT NULL,
    userid      TEXT,
    startedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finishedat  TIMESTAMPTZ,
    lastpage    SMALLINT,
    ipaddress   INET,
    devicetype  VARCHAR(20),
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chaptersession_pk PRIMARY KEY (id, startedat)
) PARTITION BY RANGE (startedat);

CREATE TABLE analytics.chaptersession_default PARTITION OF analytics.chaptersession DEFAULT;
CREATE TABLE analytics.chaptersession_2026_02 PARTITION OF analytics.chaptersession FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE analytics.chaptersession_2026_03 PARTITION OF analytics.chaptersession FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE INDEX idx_analytics_chaptersession_chapter ON analytics.chaptersession (chapterid, startedat);

COMMENT ON TABLE  analytics.chaptersession IS 'Tracks when a user opened and closed a chapter reader session. INSERT on open, UPDATE finishedat/lastpage on close or last-page event. Partitioned by startedat. Used for "time spent reading" analytics.';
COMMENT ON COLUMN analytics.chaptersession.startedat IS 'Partition key. When the user opened the chapter reader.';
COMMENT ON COLUMN analytics.chaptersession.finishedat IS 'NULL = session still open or user abandoned without triggering close event.';
COMMENT ON COLUMN analytics.chaptersession.lastpage IS 'Highest page number reached. Completion % = lastpage / chapter.pagecount.';
COMMENT ON COLUMN analytics.chaptersession.devicetype IS 'Client device class. Allowed values (Go): mobile | desktop | tablet | unknown.';

-- =============================================================================
-- END OF 60_ANALYTICS
-- =============================================================================
