# ADR 002 — Why UUIDv7

> **Status:** Accepted  
> **Date:** 2026-02-21  
> **Author:** tai.buivan.jp@gmail.com

---

## Context

Yomira needs a globally unique identifier format for primary keys on user-facing resources (users, comics, chapters, comments, etc.). The ID must be:

1. Globally unique (no central coordinator needed)
2. Safe to expose in URLs
3. Unguessable (no sequential enumeration of resources)
4. Suitable for B-tree indexes (sorted insert order reduces index fragmentation)
5. Human-debuggable (embeds creation time)

We evaluated:

| Format | Type | Considered |
|---|---|---|
| **UUIDv7** | Time-ordered UUID | ✅ Chosen |
| PostgreSQL `SERIAL` / `BIGSERIAL` | Auto-increment integer | Considered |
| UUIDv4 | Random UUID | Considered |
| ULID | Timestamp + random | Considered |
| NanoID | URL-safe random | Considered |

---

## Decision

**Use UUIDv7** for all user-facing primary keys. Use `BIGSERIAL` for internal-only high-volume tables.

---

## UUIDv7 Format

```
01952fa3-3f1e-7abc-b12e-1234567890ab
│──────────────┘│    └────────────────┘
│               │     80 bits random
48-bit Unix millisecond timestamp
                └── version 7 marker
```

- **48 bits:** Unix millisecond timestamp → time-sortable, most recent rows first
- **80 bits:** Random → collision probability negligible
- Standard UUID format (8-4-4-4-12) → works in any UUID column

---

## Reasons

### 1. Time-ordered → B-tree index friendly

UUIDv4 (random) causes **index fragmentation** — new rows insert into random B-tree pages, causing page splits and bloat. UUIDv7 (time-ordered) inserts near the end of the index like `BIGSERIAL`, same as auto-increment, maintaining a compact, sequential index.

```
UUIDv4 inserts (random):    page N → page 3 → page 7 → page 1 → page 12
UUIDv7 inserts (ordered):   page 1 → page 1 → page 1 → page 2 → page 2
```

### 2. Embeds creation time — no separate `createdat` column needed for ordering

UUIDv7 primary keys are already time-ordered, so `ORDER BY id` is equivalent to `ORDER BY createdat` for most use cases.

```sql
-- Both work correctly with UUIDv7:
SELECT * FROM social.comment WHERE comicid = $1 ORDER BY id DESC LIMIT 20;
SELECT * FROM social.comment WHERE comicid = $1 ORDER BY createdat DESC LIMIT 20;
```

### 3. Safe to expose in URLs — no enumeration

`/comics/01952fa3-3f1e-7abc-b12e-1234567890ab` — impossible to enumerate sequentially.  
`/comics/1234` — attacker can enumerate all comics by incrementing the ID.

### 4. Generated at application layer — not DB-dependent

The Go service generates UUIDv7 before inserting, which means:
- ID is known before the DB round-trip (useful for referencing in the same transaction)
- Works with any PostgreSQL version (no extension required)
- Consistent across shards or multiple DB instances

### 5. Standard UUID type — PostgreSQL native

PostgreSQL's `UUID` column type stores UUIDs as 16 bytes (efficient). UUIDv7 is valid UUID format — no special handling needed.

---

## When to use `BIGSERIAL` instead

Internal tables where IDs are never exposed to users or URLs use `BIGSERIAL` for simplicity and slightly better performance:

| Table | ID type | Reason |
|---|---|---|
| `users.session` | `BIGSERIAL` | Internal, never in URL |
| `crawler.log` | `BIGSERIAL` | High-volume, never in URL |
| `analytics.pageview` | `BIGSERIAL` | Very high volume |

---

## Tradeoffs Accepted

| Tradeoff | Mitigation |
|---|---|
| Larger than BIGINT (16 bytes vs 8 bytes) | Acceptable — UUID is industry standard |
| Slightly slower than BIGSERIAL for index writes | Negligible — ordering makes it comparable |
| Less human-readable than integers | Debug tools decode UUIDv7 timestamp |

---

## Rejected Alternatives

### PostgreSQL SERIAL / BIGSERIAL
- Sequential → exposing IDs in URLs enables resource enumeration
- Requires DB to assign ID → can't know ID before insert

### UUIDv4 (random)
- B-tree index fragmentation at scale (known issue documented extensively)
- No time information embedded
- Still widely used — acceptable but UUIDv7 is strictly better

### ULID
- Not a standard UUID format — requires custom column type or TEXT storage (larger)
- No official RFC (UUIDv7 is RFC 9562 — 2024)
- Less library support in Go/PostgreSQL ecosystem

### NanoID
- Not UUID format → incompatible with PostgreSQL `UUID` column type (use TEXT instead)
- Random-only → same B-tree fragmentation as UUIDv4
- Not time-ordered
