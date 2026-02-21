# Yomira — Entity Relationship Diagram

> Rendered natively trên GitHub via [Mermaid](https://mermaid.js.org/).  
> Schema version **1.0.0** — Updated 2026-02-21.  
> Chia thành 6 diagram theo schema để dễ đọc.

---

## Table of Contents

1. [Overview — Schema Map](#1-overview--schema-map)
2. [users — Authentication & Profiles](#2-users--authentication--profiles)
3. [core — Content Catalog](#3-core--content-catalog)
4. [library — User Shelf](#4-library--user-shelf)
5. [social — Community & Forum](#5-social--community--forum)
6. [crawler — Content Ingestion](#6-crawler--content-ingestion)
7. [analytics & system](#7-analytics--system)

---

## 1. Overview — Schema Map

High-level view of the 7 schemas and their major cross-schema relationships.

```mermaid
erDiagram
    USERS_SCHEMA["users.*"] {
        string account
        string session
        string oauthprovider
        string follow
        string readingpreference
    }

    CORE_SCHEMA["core.*"] {
        string comic
        string chapter
        string page
        string scanlationgroup
        string author_artist_tag
    }

    LIBRARY_SCHEMA["library.*"] {
        string entry
        string customlist
        string readingprogress
        string chapterread
    }

    SOCIAL_SCHEMA["social.*"] {
        string comment
        string forum_thread_post
        string notification
        string feedevent
        string report
    }

    CRAWLER_SCHEMA["crawler.*"] {
        string source
        string job
        string log
    }

    ANALYTICS_SCHEMA["analytics.*"] {
        string pageview
        string chaptersession
    }

    SYSTEM_SCHEMA["system.*"] {
        string auditlog
        string setting
        string announcement
    }

    USERS_SCHEMA ||--o{ CORE_SCHEMA       : "uploads chapters"
    USERS_SCHEMA ||--o{ LIBRARY_SCHEMA    : "has shelf"
    USERS_SCHEMA ||--o{ SOCIAL_SCHEMA     : "comments / votes"
    USERS_SCHEMA ||--o{ SYSTEM_SCHEMA     : "audit actor"
    CORE_SCHEMA  ||--o{ LIBRARY_SCHEMA    : "tracked in entry"
    CORE_SCHEMA  ||--o{ SOCIAL_SCHEMA     : "discussed"
    CORE_SCHEMA  ||--o{ CRAWLER_SCHEMA    : "crawled from"
    CORE_SCHEMA  ||--o{ ANALYTICS_SCHEMA  : "viewed"
```

---

## 2. users — Authentication & Profiles

```mermaid
erDiagram
    account {
        text       id          PK
        varchar    username    UK
        varchar    email       UK
        text       passwordhash
        varchar    role
        boolean    isverified
        boolean    isactive
        timestamptz lastloginat
        timestamptz createdat
        timestamptz deletedat
    }

    oauthprovider {
        bigint     id          PK
        text       userid      FK
        varchar    provider
        text       providerid
        varchar    email
        text       accesstoken
        timestamptz createdat
    }

    session {
        bigint     id          PK
        text       userid      FK
        text       tokenhash   UK
        varchar    devicename
        inet       ipaddress
        timestamptz expiresat
        timestamptz revokedat
    }

    follow {
        text       followerid  PK,FK
        text       followingid PK,FK
        timestamptz createdat
    }

    readingpreference {
        text       userid      PK,FK
        varchar    readingmode
        varchar    pagefit
        smallint   preloadpages
        boolean    hidensfw
        text[]     hidelanguages
    }

    account ||--o{ oauthprovider    : "linked to"
    account ||--o{ session          : "has sessions"
    account ||--o{ follow           : "follower"
    account ||--o{ follow           : "following"
    account ||--o| readingpreference : "has prefs"
```

---

## 3. core — Content Catalog

### 3a. Comic & Metadata

```mermaid
erDiagram
    comic {
        text        id             PK
        varchar     title
        text[]      titlealt
        varchar     slug           UK
        varchar     status
        varchar     contentrating
        varchar     demographic
        varchar     defaultreadmode
        varchar     originlanguage
        bigint      viewcount
        bigint      followcount
        numeric     ratingavg
        numeric     ratingbayesian
        boolean     islocked
        tsvector    searchvector
        timestamptz deletedat
    }

    author {
        integer     id     PK
        varchar     name
        text[]      namealt
    }

    artist {
        integer     id     PK
        varchar     name
        text[]      namealt
    }

    taggroup {
        integer     id     PK
        varchar     name
        varchar     slug   UK
        smallint    sortorder
    }

    tag {
        integer     id      PK
        integer     groupid FK
        varchar     name
        varchar     slug    UK
    }

    comicauthor {
        text        comicid   PK,FK
        integer     authorid  PK,FK
    }

    comicartist {
        text        comicid   PK,FK
        integer     artistid  PK,FK
    }

    comictag {
        text        comicid  PK,FK
        integer     tagid    PK,FK
    }

    comicrelation {
        text        comicid        PK,FK
        text        relatedcomicid PK,FK
        varchar     relationtype   PK
    }

    comiccover {
        text        id       PK
        text        comicid  FK
        smallint    volume
        text        imageurl
    }

    comicart {
        text        id         PK
        text        comicid    FK
        text        uploaderid FK
        varchar     arttype
        boolean     isapproved
    }

    comic       ||--o{ comicauthor   : "written by"
    comic       ||--o{ comicartist   : "drawn by"
    comic       ||--o{ comictag      : "tagged with"
    comic       ||--o{ comicrelation : "related to"
    comic       ||--o{ comiccover    : "has covers"
    comic       ||--o{ comicart      : "has art"
    author      ||--o{ comicauthor   : "authors"
    artist      ||--o{ comicartist   : "illustrates"
    tag         ||--o{ comictag      : "applied to"
    taggroup    ||--o{ tag           : "groups"
```

### 3b. Chapter & Scanlation

```mermaid
erDiagram
    comic {
        text id PK
        varchar title
    }

    scanlationgroup {
        text        id                  PK
        varchar     name
        varchar     slug                UK
        boolean     isofficialpublisher
        boolean     isactive
        timestamptz verifiedat
        text        verifiedby
    }

    scanlationgroupmember {
        text        groupid  PK,FK
        text        userid   PK,FK
        varchar     role
    }

    scanlationgroupfollow {
        text        userid   PK,FK
        text        groupid  PK,FK
    }

    chapter {
        text        id                PK
        text        comicid           FK
        integer     languageid        FK
        text        scanlationgroupid FK
        text        uploaderid        FK
        numeric     chapternumber
        varchar     syncstate
        text        externalurl
        boolean     isofficial
        boolean     islocked
        bigint      viewcount
        timestamptz publishedat
        timestamptz deletedat
    }

    page {
        text        id         PK
        text        chapterid  FK
        smallint    pagenumber
        text        imageurl
        text        imageurlhd
        varchar     format
    }

    language {
        integer     id    PK
        varchar     code  UK
        varchar     name
    }

    mediafile {
        text        id            PK
        varchar     storagebucket
        text        storagekey    UK
        char        sha256        UK
    }

    comic           ||--o{ chapter              : "has chapters"
    scanlationgroup ||--o{ chapter              : "scanlates"
    scanlationgroup ||--o{ scanlationgroupmember : "has members"
    scanlationgroup ||--o{ scanlationgroupfollow : "followed by"
    chapter         ||--o{ page                 : "has pages"
    language        ||--o{ chapter              : "translates to"
```

---

## 4. library — User Shelf

```mermaid
erDiagram
    account {
        text id PK
    }

    comic {
        text id PK
    }

    chapter {
        text id PK
    }

    entry {
        bigint      id                 PK
        text        userid             FK
        text        comicid            FK
        varchar     readingstatus
        smallint    score
        boolean     hasnew
        text        lastreadchapterid  FK
        timestamptz lastreadat
    }

    customlist {
        text        id         PK
        text        userid     FK
        varchar     name
        varchar     visibility
        timestamptz deletedat
    }

    customlistitem {
        text        listid    PK,FK
        text        comicid   PK,FK
        integer     sortorder
    }

    readingprogress {
        bigint      id         PK
        text        userid     FK
        text        comicid    FK
        text        chapterid  FK
        smallint    pagenumber
    }

    chapterread {
        bigint      id        PK
        text        userid    FK
        text        chapterid FK
        timestamptz readat
    }

    account ||--o{ entry            : "has shelf entries"
    account ||--o{ customlist       : "owns lists"
    account ||--o{ readingprogress  : "tracks progress"
    account ||--o{ chapterread      : "marks read"
    comic   ||--o{ entry            : "tracked in"
    comic   ||--o{ customlistitem   : "in lists"
    chapter ||--o{ readingprogress  : "last read"
    chapter ||--o{ chapterread      : "marked read"
    customlist ||--o{ customlistitem : "contains"
```

---

## 5. social — Community & Forum

### 5a. Comments & Ratings

```mermaid
erDiagram
    account {
        text id PK
    }

    comic {
        text id PK
    }

    chapter {
        text id PK
    }

    comicrating {
        bigint      id       PK
        text        userid   FK
        text        comicid  FK
        smallint    score
    }

    comment {
        text        id         PK
        text        userid     FK
        text        comicid    FK
        text        chapterid  FK
        text        parentid   FK
        text        body
        boolean     isdeleted
        boolean     isapproved
        integer     upvotes
        integer     downvotes
    }

    commentvote {
        text        userid    PK,FK
        text        commentid PK,FK
        smallint    vote
    }

    notification {
        text        id         PK
        text        userid     FK
        varchar     type
        varchar     entitytype
        text        entityid
        boolean     isread
    }

    comicrecommendation {
        bigint      id          PK
        text        fromcomicid FK
        text        tocomicid   FK
        text        userid      FK
        integer     upvotes
    }

    comicrecommendationvote {
        text        userid           PK,FK
        bigint      recommendationid PK,FK
        smallint    vote
    }

    feedevent {
        text        id         PK
        varchar     eventtype
        text        actorid
        varchar     entitytype
        text        entityid
        jsonb       payload
    }

    report {
        text        id         PK
        text        reporterid FK
        varchar     entitytype
        text        entityid
        varchar     reason
        varchar     status
    }

    account  ||--o{ comicrating          : "rates"
    account  ||--o{ comment              : "writes"
    account  ||--o{ commentvote          : "votes"
    account  ||--o{ notification         : "receives"
    account  ||--o{ comicrecommendation  : "recommends"
    comic    ||--o{ comicrating          : "rated in"
    comic    ||--o{ comment              : "discussed in"
    chapter  ||--o{ comment              : "discussed in"
    comment  ||--o{ comment              : "replied to"
    comment  ||--o{ commentvote          : "voted on"
    comicrecommendation ||--o{ comicrecommendationvote : "voted on"
```

### 5b. Forum

```mermaid
erDiagram
    account {
        text id PK
    }

    comic {
        text id PK
    }

    forum {
        integer     id          PK
        text        comicid     FK
        varchar     name
        varchar     slug        UK
        varchar     canpost
        integer     threadcount
        integer     postcount
    }

    forumthread {
        text        id           PK
        integer     forumid      FK
        text        authorid     FK
        varchar     title
        boolean     ispinned
        boolean     islocked
        boolean     isdeleted
        integer     replycount
        timestamptz lastpostedat
    }

    forumpost {
        text        id         PK
        text        threadid   FK
        text        authorid   FK
        text        body
        varchar     bodyformat
        boolean     isedited
        boolean     isdeleted
        integer     upvotes
        integer     downvotes
    }

    forumpostvote {
        text        userid  PK,FK
        text        postid  PK,FK
        smallint    vote
    }

    comic       ||--o| forum       : "has board"
    forum       ||--o{ forumthread : "contains"
    forumthread ||--o{ forumpost   : "has replies"
    forumpost   ||--o{ forumpostvote : "voted on"
    account     ||--o{ forumthread : "authors"
    account     ||--o{ forumpost   : "writes"
    account     ||--o{ forumpostvote : "votes"
```

---

## 6. crawler — Content Ingestion

```mermaid
erDiagram
    account {
        text id PK
    }

    comic {
        text id PK
    }

    source {
        integer     id               PK
        varchar     name             UK
        varchar     slug             UK
        text        baseurl
        varchar     extensionid
        jsonb       config
        boolean     isenabled
        smallint    consecutivefails
    }

    comicsource {
        bigint      id           PK
        text        comicid      FK
        integer     sourceid     FK
        text        sourceid_ext
        text        sourceurl
        boolean     isactive
        timestamptz lastcrawlat
    }

    job {
        text        id          PK
        integer     sourceid    FK
        text        comicid     FK
        text        triggeredby FK
        varchar     status
        integer     pagescount
        integer     errorcount
        text        lasterror
        timestamptz scheduledat
    }

    log {
        text        id       PK
        text        jobid    FK
        varchar     level
        text        message
        jsonb       meta
        timestamptz createdat
    }

    source       ||--o{ comicsource : "hosts"
    source       ||--o{ job         : "queues"
    comic        ||--o{ comicsource : "indexed in"
    comic        ||--o{ job         : "crawled by"
    job          ||--o{ log         : "logs to"
    account      ||--o{ job         : "triggers"
```

---

## 7. analytics & system

```mermaid
erDiagram
    account {
        text id PK
    }

    pageview {
        bigint      id         PK
        varchar     entitytype
        text        entityid
        text        userid
        inet        ipaddress
        timestamptz createdat
    }

    chaptersession {
        bigint      id         PK
        text        chapterid
        text        userid
        timestamptz startedat
        timestamptz finishedat
        smallint    lastpage
        varchar     devicetype
    }

    auditlog {
        text        id         PK
        text        actorid    FK
        varchar     action
        varchar     entitytype
        text        entityid
        jsonb       before
        jsonb       after
        timestamptz createdat
    }

    setting {
        integer     id    PK
        varchar     key   UK
        text        value
        text        description
    }

    announcement {
        text        id          PK
        text        authorid    FK
        varchar     title
        text        body
        varchar     bodyformat
        boolean     ispublished
        boolean     ispinned
        timestamptz expiresat
        timestamptz deletedat
    }

    account ||--o{ pageview       : "generates"
    account ||--o{ chaptersession : "reads"
    account ||--o{ auditlog       : "audited"
    account ||--o{ announcement   : "authors"
```

---

## Key Conventions

| Symbol | Meaning |
|---|---|
| `PK` | Primary Key |
| `FK` | Foreign Key |
| `UK` | Unique constraint |
| `\|\|--o{` | One to zero-or-many |
| `\|\|--o\|` | One to zero-or-one |
| `\|\|--\|{` | One to one-or-many |

> **Note:** `analytics.pageview` and `analytics.chaptersession` do not have FK constraints on `userid` and `chapterid` to avoid partition overhead. Referential integrity is ensured by the application layer.
