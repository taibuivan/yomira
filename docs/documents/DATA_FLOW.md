# Data Flow — Yomira End-to-End

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

This document explains how data flows through the Yomira system from ingestion to end-user display — covering the crawler pipeline, read path, write path, and event-driven updates.

---

## Table of Contents

1. [System Data Map](#1-system-data-map)
2. [Crawler Pipeline (Ingestion)](#2-crawler-pipeline-ingestion)
3. [Comic Read Path](#3-comic-read-path)
4. [Chapter Reader Path](#4-chapter-reader-path)
5. [User Write Path](#5-user-write-path)
6. [Social Event Flow](#6-social-event-flow)
7. [Analytics Pipeline](#7-analytics-pipeline)
8. [Background Job Flow](#8-background-job-flow)
9. [Denormalization Strategy](#9-denormalization-strategy)

---

## 1. System Data Map

```
External Sources (MangaDex, MangaPlus, etc.)
        │
        ▼
  [50_CRAWLER]                  ← crawl.source, crawl.job, crawl.log
        │
        │ creates/updates
        ▼
  [20_CORE]                     ← comic, chapter, page, author, artist, tag...
        │
        ├──────────────────────────────────────────────────────┐
        │                                                      │
        ▼                                                      ▼
  [30_LIBRARY]                  ← entry, chapterread, list    [40_SOCIAL]   ← rating, comment, notification
  [10_USERS]                    ← account, session, follow    [70_SYSTEM]   ← auditlog, setting
        │                                                      │
        └──────────────────┬───────────────────────────────────┘
                           │
                           ▼
                    [60_ANALYTICS]          ← pageview, chaptersession
                           │
                           ▼
                    Admin Dashboard
```

---

## 2. Crawler Pipeline (Ingestion)

```
External Site (e.g. MangaDex API)
        │
        │  HTTP GET /manga/{id}, /chapter/{id}
        ▼
  crawler.job (status: running)
        │
        │ parse + normalize data
        ▼
  ┌─────────────────────────────────────────────────────┐
  │  Upsert into core.*                                 │
  │                                                     │
  │  1. core.language        ← BCP-47 code, name       │
  │  2. core.author          ← name, alt names         │
  │  3. core.artist          ← name, alt names         │
  │  4. core.tag             ← slug, group             │
  │  5. core.comic           ← title, slug, status,    │
  │                             demographics, rating   │
  │  6. core.comicauthor     ← M:N link                │
  │  7. core.comictag        ← M:N link                │
  │  8. core.chapter         ← number, lang, group     │
  │  9. core.page            ← pagenumber, imageurl    │
  └─────────────────────────────────────────────────────┘
        │
        │ on new chapter
        ▼
  ┌─────────────────────────────────────────────────────┐
  │  Fan-out notifications                              │
  │  SELECT userid FROM library.entry                  │
  │  WHERE comicid = $1 AND notify_new_chapter = TRUE  │
  │  → INSERT INTO social.notification (type: new_chapter)│
  │  → INSERT INTO social.feedevent (type: chapter_published)│
  │  → Queue email (if mail prefs enabled)             │
  └─────────────────────────────────────────────────────┘
        │
        │ update job
        ▼
  crawler.job (status: completed, numupdated: N)
  crawler.log (N log lines with timing)
```

### Deduplication

```sql
-- Comics deduplicated by externalid + sourceid:
INSERT INTO crawler.comicsource (comicid, sourceid, externalid, sourcetitle)
VALUES ($1, $2, $3, $4)
ON CONFLICT (sourceid, externalid) DO UPDATE
    SET sourcetitle = EXCLUDED.sourcetitle, updatedat = NOW();

-- Chapters deduplicated by comicid + chapternumber + languageid + groupid:
INSERT INTO core.chapter (id, comicid, chapternumber, languageid, groupid, ...)
ON CONFLICT (comicid, chapternumber, languageid, groupid) DO UPDATE
    SET title = EXCLUDED.title, updatedat = NOW();
```

---

## 3. Comic Read Path

```
Client: GET /api/v1/comics/:id

  ┌─────────────────────────────────────────────────────┐
  │  Redis cache check                                  │
  │  key: "comic:{id}"  TTL: 60s                       │
  └──────────┬──────────────────────────────────────────┘
             │ cache miss
             ▼
  ┌─────────────────────────────────────────────────────┐
  │  PostgreSQL query                                   │
  │                                                     │
  │  SELECT c.*, array_agg(DISTINCT a.name) AS authors,│
  │         array_agg(DISTINCT t.slug) AS tags,        │
  │         (SELECT score FROM social.comicrating       │
  │          WHERE userid = $caller_id) AS userrating  │
  │  FROM core.comic c                                  │
  │  LEFT JOIN core.comicauthor ca ON ca.comicid = c.id│
  │  LEFT JOIN core.author a ON a.id = ca.authorid     │
  │  LEFT JOIN core.comictag ct ON ct.comicid = c.id   │
  │  LEFT JOIN core.tag t ON t.id = ct.tagid           │
  │  WHERE c.id = $1 AND c.deletedat IS NULL           │
  │  GROUP BY c.id                                     │
  └──────────┬──────────────────────────────────────────┘
             │
             ▼
  JSON response → cached in Redis 60s → Client

  Side effect (async):
  POST analytics.pageview (entitytype: "comic", comicid: $id)
```

### Personalized fields (authenticated requests only)

```
userrating      ← social.comicrating.score WHERE userid = caller
isfollowing     ← library.entry.comicid WHERE userid = caller
readingstatus   ← library.entry.readingstatus WHERE userid = caller
lastreadchapter ← library.chapterread + core.chapter JOIN
```

---

## 4. Chapter Reader Path

```
Client: GET /api/v1/chapters/:id

  ┌─────────────────────────────────────────────────────┐
  │  Fetch chapter + all pages in one query             │
  │                                                     │
  │  SELECT ch.*, array_agg(               │
  │      json_build_object(                │
  │          'pagenumber', p.pagenumber,   │
  │          'imageurl', p.imageurl)       │
  │      ORDER BY p.pagenumber             │
  │  ) FILTER (WHERE p.id IS NOT NULL) AS pages        │
  │  FROM core.chapter ch                               │
  │  LEFT JOIN core.page p ON p.chapterid = ch.id      │
  │  WHERE ch.id = $1 AND ch.deletedat IS NULL         │
  │  GROUP BY ch.id                                     │
  └──────────┬──────────────────────────────────────────┘
             │
             ▼
  Client receives chapter + pages array
             │
             │  User reads pages
             ▼
  ┌─────────────────────────────────────────────────────┐
  │  On page view (batched, sent every 30 pages or end) │
  │  POST /analytics/page-view                          │
  │  → INSERT analytics.pageview (chapterid, comicid,  │
  │      pagenumber, sessionid, userid)                 │
  └──────────────────────────────────────────────────────┘
             │
             │  On 80% completion
             ▼
  ┌─────────────────────────────────────────────────────┐
  │  POST /chapters/:id/read                            │
  │  → INSERT library.chapterread (userid, chapterid)  │
  │  → UPDATE library.entry SET lastchapterreadat,     │
  │          lastreadchapternumber = ch.chapternumber   │
  │  → UPDATE library.entry SET hasnew = FALSE         │
  │      IF no unread chapters remain                  │
  └──────────────────────────────────────────────────────┘
```

---

## 5. User Write Path

### Rating a comic

```
Client: PUT /comics/:id/rating  { score: 9 }

  Service:
  1. Validate score ∈ [1, 10]
  2. UPSERT social.comicrating (userid, comicid, score)
  3. Recalculate ratingavg + ratingbayesian on core.comic
       → See QUERY_PATTERNS.md §6 for Bayesian formula
  4. Return updated rating stats

  Response: { data: { score: 9, ratingavg: 9.07, ratingbayesian: 9.01, ratingcount: 58422 } }
```

### Following a comic

```
Client: POST /library/:comicid/follow

  Service:
  1. INSERT library.entry (userid, comicid, readingstatus: 'plan_to_read')
       ON CONFLICT DO NOTHING  (already following)
  2. UPDATE core.comic SET followcount = followcount + 1

  Side effect (async):
  → INSERT social.feedevent (eventtype: 'comic_followed') for followers of caller
```

### Adding a comment

```
Client: POST /comics/:id/comments  { body: "..." }

  Service + Storage:
  1. Validate body length (1–10,000 chars)
  2. Check caller.isverified = TRUE
  3. INSERT social.comment (userid, comicid, body)
       isapproved = (caller.totalcomments >= 5)  ← moderation queue for new users
  4. If isapproved = TRUE:
       → INSERT social.notification for comic author (if they have notifications enabled)
       → INSERT social.feedevent for followers

  Response 201: Comment object
```

---

## 6. Social Event Flow

### Notification fan-out (on new chapter)

```
Event: new chapter crawled for comic X

  1. Find subscribers:
     SELECT userid FROM library.entry
     WHERE comicid = $comicId AND readingstatus != 'dropped'

  2. For each subscriber:
     INSERT social.notification (userid, type: 'new_chapter',
         entitytype: 'chapter', entityid: $chapterId)

  3. Update library.entry.hasnew = TRUE for all subscribers

  4. Queue email (if user.emailpref.new_chapter = TRUE):
     → Send via mail worker (see batch jobs)

  5. INSERT social.feedevent (type: 'chapter_published', entityid: $chapterId)
     for followers of the scanlation group
```

### Follow graph propagation

```
User A follows User B:
  → INSERT users.follow (followerid: A, followingid: B)
  → INSERT social.notification to B (type: 'follow', actor: A)
  → INSERT social.feedevent (type: 'user_followed') for A's followers
     → appears in followers' /me/feed
```

---

## 7. Analytics Pipeline

```
Real-time event (per page view):
  Client → POST /api/v1/analytics/page-view (internal endpoint)
         → INSERT analytics.pageview (partitioned by eventdate month)

Background flush (every 5 minutes, batch job):
  1. SELECT SUM(views) FROM analytics.pageview
     WHERE eventdate >= NOW() - INTERVAL '5 min'
     GROUP BY comicid, chapterid

  2. UPDATE core.comic SET viewcount = viewcount + $delta_views
     UPDATE core.chapter SET viewcount = viewcount + $delta_views

  Result: comic.viewcount is eventually consistent (max 5 min lag)
  Exact view counts always available from analytics.pageview
```

### Analytics data access pattern

```
Admin dashboard: SELECT ... FROM analytics.pageview
    WHERE eventdate >= '2026-02-01' AND eventdate < '2026-03-01'
    Group by day, comic, chapter

→ PostgreSQL partition pruning: only scans 1 month's partition (fast)
→ Results cached in Redis for 5 min (admin dashboard)
```

---

## 8. Background Job Flow

```
Scheduler (cron-like Go ticker):
  Every minute:
    SELECT id FROM crawler.job
    WHERE status = 'pending'
      AND scheduledat <= NOW()
    ORDER BY scheduledat ASC
    LIMIT 5
    FOR UPDATE SKIP LOCKED   ← prevents double-execution on multiple instances

  For each job:
    UPDATE crawler.job SET status = 'running', startedat = NOW()
    → spawn goroutine → crawl → update status = 'completed'|'failed'
    → write crawler.log entries
```

### Batch job triggers

```
POST /admin/batch/{job} → validates admin role
    → acquires Redis distributed lock (prevent duplicate run)
    → runs job in goroutine
    → streams progress to database (system.auditlog)
    → releases lock on completion/error
```

---

## 9. Denormalization Strategy

Several counts are **denormalized** (stored on the parent row) for query performance:

| Denormalized column | Source | Update trigger |
|---|---|---|
| `core.comic.followcount` | COUNT of `library.entry` | On follow/unfollow (delta ±1) |
| `core.comic.viewcount` | SUM of `analytics.pageview` | Batch flush every 5 min |
| `core.comic.ratingavg` | AVG of `social.comicrating.score` | On each rating upsert |
| `core.comic.ratingbayesian` | Bayesian formula | On each rating upsert |
| `core.comic.ratingcount` | COUNT of `social.comicrating` | On each rating upsert |
| `core.comic.chaptercount` | COUNT of `core.chapter` | On chapter create/delete |
| `social.comment.upvotes` | COUNT of `social.commentvote` | On vote (delta ±1) |
| `social.comment.replycount` | COUNT of child comments | On reply insert |
| `social.forumthread.replycount` | COUNT of `social.forumpost` | On post (delta ±1) |

**Drift correction:** A nightly batch job recalculates exact counts and applies corrections. Delta updates under concurrent load can have slight drift — the nightly job ensures correctness.

```sql
-- Nightly recalculation (batch job: core.recalc_follower_counts)
UPDATE core.comic
SET followcount = (
    SELECT COUNT(*) FROM library.entry WHERE comicid = core.comic.id
)
WHERE deletedat IS NULL;
```
