-- =============================================================================
-- YOMIRA — 40_SOCIAL
-- Schema: social
-- @author  tai.buivan.jp@gmail.com
-- Tables (12):
--   comicrating · comment · commentvote · notification
--   comicrecommendation · comicrecommendationvote
--   feedevent
--   forum · forumthread · forumpost · forumpostvote
--   report
-- Version : 3.0.0  |  Updated : 2026-02-21
-- Depends : 00_SETUP/SETUP.sql · 10_USERS/USERS.sql · 20_CORE/CORE.sql
-- =============================================================================

-- ============================================================
-- social.comicrating  — Public 1–10 score per user per comic
-- ============================================================
CREATE TABLE social.comicrating (
    id        BIGSERIAL   NOT NULL,
    userid    TEXT        NOT NULL,
    comicid   TEXT        NOT NULL,
    score     SMALLINT    NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comicrating_pkey     PRIMARY KEY (id),
    CONSTRAINT comicrating_uq       UNIQUE (userid, comicid),
    CONSTRAINT comicrating_user_fk  FOREIGN KEY (userid)  REFERENCES users.account (id) ON DELETE CASCADE,
    CONSTRAINT comicrating_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic    (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'comicrating');
CREATE INDEX idx_social_comicrating_comicid ON social.comicrating (comicid);
COMMENT ON TABLE  social.comicrating IS 'Public 1–10 rating per (user, comic). One row, upserted on change. Aggregated into core.comic.ratingavg and ratingbayesian.';
COMMENT ON COLUMN social.comicrating.score IS 'Integer 1–10. Range enforced by Go service, not by DB constraint.';

-- ============================================================
-- social.comment  — Threaded comments on comics or chapters
-- ============================================================
-- Attached to EITHER a comic OR a chapter — never both.
-- Mutual exclusivity (comicid XOR chapterid) enforced by Go service layer.
-- Soft-deleted: isdeleted=TRUE keeps row for thread continuity.
-- ============================================================
CREATE TABLE social.comment (
    id          TEXT        NOT NULL,
    userid      TEXT        NOT NULL,
    comicid     TEXT,
    chapterid   TEXT,
    parentid    TEXT,
    body        TEXT        NOT NULL,
    isedited    BOOLEAN     NOT NULL DEFAULT FALSE,
    isdeleted   BOOLEAN     NOT NULL DEFAULT FALSE,
    isapproved  BOOLEAN     NOT NULL DEFAULT TRUE,
    upvotes     INTEGER     NOT NULL DEFAULT 0,
    downvotes   INTEGER     NOT NULL DEFAULT 0,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comment_pkey      PRIMARY KEY (id),
    CONSTRAINT comment_user_fk   FOREIGN KEY (userid)    REFERENCES users.account  (id) ON DELETE CASCADE,
    CONSTRAINT comment_comic_fk  FOREIGN KEY (comicid)   REFERENCES core.comic    (id) ON DELETE CASCADE,
    CONSTRAINT comment_chap_fk   FOREIGN KEY (chapterid) REFERENCES core.chapter  (id) ON DELETE CASCADE,
    CONSTRAINT comment_parent_fk FOREIGN KEY (parentid)  REFERENCES social.comment (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'comment');
CREATE INDEX idx_social_comment_comicid   ON social.comment (comicid,   createdat DESC) WHERE comicid   IS NOT NULL AND isdeleted = FALSE;
CREATE INDEX idx_social_comment_chapterid ON social.comment (chapterid, createdat DESC) WHERE chapterid IS NOT NULL AND isdeleted = FALSE;
CREATE INDEX idx_social_comment_parentid  ON social.comment (parentid)                  WHERE parentid  IS NOT NULL;

COMMENT ON TABLE  social.comment IS 'Threaded comment attached to a comic or a chapter (Go enforces XOR, never both). parentid = NULL means top-level; non-NULL means reply to parent. Soft-deleted: isdeleted=TRUE preserves row for thread continuity in UI.';
COMMENT ON COLUMN social.comment.comicid IS 'Set when comment is on a comic page. Null when on a chapter. Go enforces XOR.';
COMMENT ON COLUMN social.comment.isapproved IS 'FALSE = held for moderation (new user first posts, spam detection).';
COMMENT ON COLUMN social.comment.upvotes IS 'Denormalized counter. Incremented when a commentvote with vote=1 is inserted.';

-- ============================================================
-- social.commentvote  — Up/down vote per user per comment
-- ============================================================
CREATE TABLE social.commentvote (
    userid    TEXT        NOT NULL,
    commentid TEXT        NOT NULL,
    vote      SMALLINT    NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT commentvote_pkey       PRIMARY KEY (userid, commentid),
    CONSTRAINT commentvote_user_fk    FOREIGN KEY (userid)    REFERENCES users.account  (id) ON DELETE CASCADE,
    CONSTRAINT commentvote_comment_fk FOREIGN KEY (commentid) REFERENCES social.comment  (id) ON DELETE CASCADE
);
CREATE INDEX idx_social_commentvote_commentid ON social.commentvote (commentid);
COMMENT ON TABLE  social.commentvote IS 'One vote per (user, comment). Drives comment.upvotes / downvotes counters.';
COMMENT ON COLUMN social.commentvote.vote IS 'Vote direction (Go): 1 = upvote, -1 = downvote.';

-- ============================================================
-- social.notification  — System-to-user notification inbox
-- ============================================================
CREATE TABLE social.notification (
    id         TEXT         NOT NULL,
    userid     TEXT         NOT NULL,
    type       VARCHAR(30)  NOT NULL,
    title      VARCHAR(300) NOT NULL,
    body       TEXT,
    entitytype VARCHAR(50),
    entityid   TEXT,
    isread     BOOLEAN      NOT NULL DEFAULT FALSE,
    createdat  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_pkey    PRIMARY KEY (id),
    CONSTRAINT notification_user_fk FOREIGN KEY (userid) REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'notification');
CREATE INDEX idx_social_notification_userid ON social.notification (userid, isread, createdat DESC);
CREATE INDEX idx_social_notification_unread ON social.notification (userid) WHERE isread = FALSE;
COMMENT ON TABLE  social.notification IS 'Pull-based notification inbox. Go writes on events; user polls /notifications.';
COMMENT ON COLUMN social.notification.type IS 'Event type (Go): new_chapter | system | announcement | comment_reply | follow.';
COMMENT ON COLUMN social.notification.entitytype IS 'Type of linked entity (Go): comic | chapter | comment | user.';
COMMENT ON COLUMN social.notification.entityid IS
    'UUID/ID of the linked entity for deep-linking from the notification.';

-- ============================================================
-- social.comicrecommendation  — "Recommendations" tab
-- ============================================================
-- User A says: "If you like comic X, you should also read comic Y."
-- fromcomicid = comic being viewed (X), tocomicid = recommended comic (Y).
-- Self-recommendation (fromcomicid = tocomicid) prevented by Go service.
-- ============================================================
CREATE TABLE social.comicrecommendation (
    id          BIGSERIAL   NOT NULL,
    fromcomicid TEXT        NOT NULL,
    tocomicid   TEXT        NOT NULL,
    userid      TEXT        NOT NULL,
    reason      TEXT,
    upvotes     INTEGER     NOT NULL DEFAULT 0,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comicrecommendation_pkey    PRIMARY KEY (id),
    CONSTRAINT comicrecommendation_uq      UNIQUE (fromcomicid, tocomicid, userid),
    CONSTRAINT comicrecommendation_from_fk FOREIGN KEY (fromcomicid) REFERENCES core.comic    (id) ON DELETE CASCADE,
    CONSTRAINT comicrecommendation_to_fk   FOREIGN KEY (tocomicid)   REFERENCES core.comic    (id) ON DELETE CASCADE,
    CONSTRAINT comicrecommendation_user_fk FOREIGN KEY (userid)      REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'comicrecommendation');
CREATE INDEX idx_social_comicrecommendation_from ON social.comicrecommendation (fromcomicid, upvotes DESC);
CREATE INDEX idx_social_comicrecommendation_to   ON social.comicrecommendation (tocomicid);
COMMENT ON TABLE  social.comicrecommendation IS 'User-submitted recommendation: "readers of X should also read Y". Shown on the Recommendations tab of a comic, sorted by upvotes.';
COMMENT ON COLUMN social.comicrecommendation.fromcomicid IS 'Source comic (currently being viewed by the user).';
COMMENT ON COLUMN social.comicrecommendation.tocomicid IS 'Recommended comic. Go prevents fromcomicid = tocomicid.';
COMMENT ON COLUMN social.comicrecommendation.upvotes IS 'Denormalized upvote count. Incremented by comicrecommendationvote inserts.';

-- ============================================================
-- social.comicrecommendationvote  — Votes on recommendations
-- ============================================================
CREATE TABLE social.comicrecommendationvote (
    userid           TEXT        NOT NULL,
    recommendationid BIGINT      NOT NULL,
    vote             SMALLINT    NOT NULL,
    createdat        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT comicrecvote_pkey   PRIMARY KEY (userid, recommendationid),
    CONSTRAINT comicrecvote_usr_fk FOREIGN KEY (userid)           REFERENCES users.account              (id) ON DELETE CASCADE,
    CONSTRAINT comicrecvote_rec_fk FOREIGN KEY (recommendationid) REFERENCES social.comicrecommendation (id) ON DELETE CASCADE
);
COMMENT ON TABLE  social.comicrecommendationvote IS 'Upvote/downvote on a recommendation. Drives comicrecommendation.upvotes counter.';
COMMENT ON COLUMN social.comicrecommendationvote.vote IS 'Vote direction (Go): 1 = upvote, -1 = downvote.';

-- ============================================================
-- social.feedevent  — Activity Feed events
-- ============================================================
-- Pull model: queried by the /feed endpoint filtering on followed entity IDs.
-- Push model (for >1M users): fan-out to Redis sorted set per user on publish.
-- ============================================================
CREATE TABLE social.feedevent (
    id         TEXT        NOT NULL,
    eventtype  VARCHAR(30) NOT NULL,
    actorid    TEXT,
    entitytype VARCHAR(20) NOT NULL,
    entityid   TEXT        NOT NULL,
    payload    JSONB       NOT NULL DEFAULT '{}',
    createdat  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT feedevent_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_social_feedevent_entity    ON social.feedevent (entitytype, entityid);
CREATE INDEX idx_social_feedevent_createdat ON social.feedevent (createdat);
COMMENT ON TABLE  social.feedevent IS 'Activity stream events for the Feed tab. Pull model: SELECT WHERE entityid IN (followed_comic_ids UNION followed_group_ids) ORDER BY createdat DESC. For >1M users, replace with Redis sorted-set push model.';
COMMENT ON COLUMN social.feedevent.eventtype IS 'Event type (Go): chapter_published | comic_followed | user_followed | group_chapter | list_updated | comic_updated.';
COMMENT ON COLUMN social.feedevent.payload IS 'Denormalized JSONB for rendering feed items without extra JOINs. Schema is eventtype-specific; defined by Go structs.';
COMMENT ON COLUMN social.feedevent.actorid IS 'User who triggered the event. NULL = system-generated event.';

-- ============================================================
-- social.forum  — Forum board
-- ============================================================
-- NULL comicid = site-wide board.
-- Non-NULL comicid = per-comic discussion board (auto-created on comic add).
-- ============================================================
CREATE SEQUENCE social.forum_id_seq START 100000 INCREMENT 1;
CREATE TABLE social.forum (
    id          INTEGER      NOT NULL DEFAULT nextval('social.forum_id_seq'),
    comicid     TEXT,
    name        VARCHAR(200) NOT NULL,
    slug        VARCHAR(220) NOT NULL,
    description TEXT,
    sortorder   SMALLINT     NOT NULL DEFAULT 0,
    isarchived  BOOLEAN      NOT NULL DEFAULT FALSE,
    canpost     VARCHAR(20)  NOT NULL DEFAULT 'member',
    threadcount INTEGER      NOT NULL DEFAULT 0,
    postcount   INTEGER      NOT NULL DEFAULT 0,
    createdat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT forum_pkey     PRIMARY KEY (id),
    CONSTRAINT forum_slug_uq  UNIQUE (slug),
    CONSTRAINT forum_comic_fk FOREIGN KEY (comicid) REFERENCES core.comic (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'forum');
CREATE INDEX idx_social_forum_comicid ON social.forum (comicid) WHERE comicid IS NOT NULL;
COMMENT ON TABLE  social.forum IS 'Forum board. NULL comicid = site-wide (General, Announcements, Support). Non-NULL = per-comic board auto-created when a comic is added.';
COMMENT ON COLUMN social.forum.canpost IS 'Minimum role to post in this board (Go): admin | moderator | member.';
COMMENT ON COLUMN social.forum.threadcount IS 'Denormalized. Updated on thread create/soft-delete.';
COMMENT ON COLUMN social.forum.postcount IS 'Denormalized total posts across all threads. Updated on post create/soft-delete.';

-- ============================================================
-- social.forumthread  — Discussion thread within a board
-- ============================================================
CREATE TABLE social.forumthread (
    id           TEXT         NOT NULL,
    forumid      INTEGER      NOT NULL,
    authorid     TEXT         NOT NULL,
    title        VARCHAR(500) NOT NULL,
    ispinned     BOOLEAN      NOT NULL DEFAULT FALSE,
    islocked     BOOLEAN      NOT NULL DEFAULT FALSE,
    isdeleted    BOOLEAN      NOT NULL DEFAULT FALSE,
    replycount   INTEGER      NOT NULL DEFAULT 0,
    viewcount    INTEGER      NOT NULL DEFAULT 0,
    lastpostedat TIMESTAMPTZ,
    lastposterid TEXT,
    createdat    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updatedat    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT forumthread_pkey      PRIMARY KEY (id),
    CONSTRAINT forumthread_forum_fk  FOREIGN KEY (forumid)  REFERENCES social.forum  (id) ON DELETE CASCADE,
    CONSTRAINT forumthread_author_fk FOREIGN KEY (authorid) REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'forumthread');
CREATE INDEX idx_social_forumthread_forumid  ON social.forumthread (forumid, lastpostedat DESC) WHERE isdeleted = FALSE;
CREATE INDEX idx_social_forumthread_pinned   ON social.forumthread (forumid)                    WHERE ispinned = TRUE AND isdeleted = FALSE;
CREATE INDEX idx_social_forumthread_authorid ON social.forumthread (authorid);
COMMENT ON TABLE  social.forumthread IS 'Discussion thread within a forum board.';
COMMENT ON COLUMN social.forumthread.ispinned IS 'TRUE = always shown at the top of thread list regardless of activity timestamp.';
COMMENT ON COLUMN social.forumthread.islocked IS 'TRUE = thread closed; no new replies accepted.';
COMMENT ON COLUMN social.forumthread.lastpostedat IS 'Updated on each new reply. Used for "sort by latest activity" query.';
COMMENT ON COLUMN social.forumthread.replycount IS 'Denormalized reply count (excludes soft-deleted posts). Updated on post change.';

-- ============================================================
-- social.forumpost  — Individual reply within a thread
-- ============================================================
CREATE TABLE social.forumpost (
    id          TEXT        NOT NULL,
    threadid    TEXT        NOT NULL,
    authorid    TEXT        NOT NULL,
    body        TEXT        NOT NULL,
    bodyformat  VARCHAR(10) NOT NULL DEFAULT 'markdown',
    isedited    BOOLEAN     NOT NULL DEFAULT FALSE,
    isdeleted   BOOLEAN     NOT NULL DEFAULT FALSE,
    isapproved  BOOLEAN     NOT NULL DEFAULT TRUE,
    upvotes     INTEGER     NOT NULL DEFAULT 0,
    downvotes   INTEGER     NOT NULL DEFAULT 0,
    createdat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT forumpost_pkey      PRIMARY KEY (id),
    CONSTRAINT forumpost_thread_fk FOREIGN KEY (threadid) REFERENCES social.forumthread (id) ON DELETE CASCADE,
    CONSTRAINT forumpost_author_fk FOREIGN KEY (authorid) REFERENCES users.account      (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'forumpost');
CREATE INDEX idx_social_forumpost_threadid ON social.forumpost (threadid, createdat) WHERE isdeleted = FALSE;
CREATE INDEX idx_social_forumpost_authorid ON social.forumpost (authorid);
COMMENT ON TABLE  social.forumpost IS 'Individual post/reply in a forum thread. Soft-deleted via isdeleted.';
COMMENT ON COLUMN social.forumpost.bodyformat IS 'Render format (Go): markdown | plain.';
COMMENT ON COLUMN social.forumpost.isedited IS 'TRUE = shows "edited" label in UI after original post is modified.';
COMMENT ON COLUMN social.forumpost.isapproved IS 'FALSE = held for moderation review before displaying to other users.';

-- ============================================================
-- social.forumpostvote  — Up/down vote on forum posts
-- ============================================================
CREATE TABLE social.forumpostvote (
    userid    TEXT        NOT NULL,
    postid    TEXT        NOT NULL,
    vote      SMALLINT    NOT NULL,
    createdat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT forumpostvote_pkey     PRIMARY KEY (userid, postid),
    CONSTRAINT forumpostvote_user_fk  FOREIGN KEY (userid) REFERENCES users.account    (id) ON DELETE CASCADE,
    CONSTRAINT forumpostvote_post_fk  FOREIGN KEY (postid) REFERENCES social.forumpost (id) ON DELETE CASCADE
);
CREATE INDEX idx_social_forumpostvote_postid ON social.forumpostvote (postid);
COMMENT ON TABLE  social.forumpostvote IS 'Vote per (user, forumpost). Drives forumpost.upvotes/downvotes counters.';
COMMENT ON COLUMN social.forumpostvote.vote IS 'Vote direction (Go): 1 = upvote, -1 = downvote.';

-- ============================================================
-- social.report  — Content moderation reports
-- ============================================================
CREATE TABLE social.report (
    id         TEXT        NOT NULL,
    reporterid TEXT        NOT NULL,
    entitytype VARCHAR(30) NOT NULL,
    entityid   TEXT        NOT NULL,
    reason     VARCHAR(50) NOT NULL,
    details    TEXT,
    status     VARCHAR(20) NOT NULL DEFAULT 'open',
    resolvedby TEXT,
    resolvedat TIMESTAMPTZ,
    resolution TEXT,
    createdat  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updatedat  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_pkey        PRIMARY KEY (id),
    CONSTRAINT report_reporter_fk FOREIGN KEY (reporterid) REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('social', 'report');
CREATE INDEX idx_social_report_status     ON social.report (status) WHERE status IN ('open','reviewing');
CREATE INDEX idx_social_report_entitytype ON social.report (entitytype, entityid);
CREATE INDEX idx_social_report_reporterid ON social.report (reporterid);
COMMENT ON TABLE  social.report IS 'User-submitted content reports fed into the moderation queue. Moderators see reports filtered by status = open/reviewing.';
COMMENT ON COLUMN social.report.entitytype IS 'What is being reported (Go): comic | chapter | comment | forumpost | user | scanlationgroup.';
COMMENT ON COLUMN social.report.reason IS 'Report category (Go): spam | violence | explicit_content | misinformation | copyright | duplicate | low_quality | other.';
COMMENT ON COLUMN social.report.status IS 'Resolution state (Go): open | reviewing | resolved | dismissed.';
COMMENT ON COLUMN social.report.resolution IS 'Moderator''s internal note explaining their decision.';

-- =============================================================================
-- SEED: Default forum boards
-- =============================================================================
INSERT INTO social.forum (name, slug, description, sortorder, canpost) VALUES
    ('Announcements',     'announcements', 'Official Yomira announcements.',                  0, 'admin'),
    ('General',           'general',       'Off-topic chat and general conversation.',         1, 'member'),
    ('Support',           'support',       'Help, bug reports, and feedback.',                 2, 'member'),
    ('Manga Discussion',  'manga',         'Discuss specific titles and chapters.',            3, 'member'),
    ('Scanlation Groups', 'groups',        'Recruitment and discussion for translator teams.', 4, 'member')
ON CONFLICT (slug) DO NOTHING;

-- =============================================================================
-- END OF 40_SOCIAL
-- =============================================================================
