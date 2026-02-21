# Frontend Guide — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Status:** Phase 2 (planned)  
> **Stack:** Go + html/template (SSR) with progressive enhancement, or Next.js (TBD)

---

## Table of Contents

1. [Architecture Decision](#1-architecture-decision)
2. [Page Inventory](#2-page-inventory)
3. [URL Structure](#3-url-structure)
4. [API Client Pattern](#4-api-client-pattern)
5. [Authentication Flow (Client)](#5-authentication-flow-client)
6. [Reader Modes](#6-reader-modes)
7. [Design System Tokens](#7-design-system-tokens)
8. [State Management](#8-state-management)
9. [Performance Targets](#9-performance-targets)

---

## 1. Architecture Decision

> **Decision pending** — two options under evaluation:

### Option A: Go SSR + HTMX (lightweight)
- Server renders HTML with `html/template`
- HTMX for interactive updates (no full-page reload)
- Alpine.js for client-side state (reader preferences)
- **Pros:** Simpler deployment (single binary), great SEO out of the box
- **Cons:** Reader experience limited without JS framework; complex reader UI harder to build

### Option B: Next.js (App Router) + Go API
- Next.js for SSR/SSG of catalog pages
- Client-side reading with React (reader component)
- **Pros:** Best reading UX, large ecosystem
- **Cons:** Two services to deploy; adds Node.js dependency

**Current lean:** Option B for Phase 2, as the manga reader UX is complex enough to warrant a full SPA component.

---

## 2. Page Inventory

### Public pages (SSR — SEO critical)

| Page | URL | SSR | Cache |
|---|---|---|---|
| Home / Discover | `/` | Yes | CDN 60s |
| Comic detail | `/comics/:slug` | Yes | CDN 30s |
| Chapter reader | `/comics/:slug/:chapter` | No (CSR) | — |
| Author page | `/authors/:id` | Yes | CDN 5min |
| Artist page | `/artists/:id` | Yes | CDN 5min |
| Tag browse | `/tags/:slug` | Yes | CDN 5min |
| Group page | `/groups/:id` | Yes | CDN 5min |
| Search results | `/search?q=...` | No (CSR) | — |
| Forum board | `/forum/:slug` | Yes | CDN 30s |
| Forum thread | `/forum/:slug/:threadId` | Yes | CDN 30s |

### Authenticated pages (CSR — no SSR needed)

| Page | URL | Notes |
|---|---|---|
| My Library | `/library` | Reading shelf |
| Reading History | `/history` | Recent chapters read |
| My Lists | `/lists`, `/lists/:id` | Custom lists |
| Notifications | `/notifications` | — |
| Activity Feed | `/feed` | Following activity |
| Settings | `/settings` | Profile, preferences, security |
| Session Management | `/settings/sessions` | Active login sessions |
| Email Preferences | `/settings/email` | Notification subscriptions |

### Admin pages (CSR — admin role only)

| Page | URL | MSD | Notes |
|---|---|---|---|
| User management | `/admin/users` | — | — |
| Comic management | `/admin/comics` | — | CRUD |
| Chapter management | `/admin/chapters` | — | Sync state |
| Crawler jobs | `/admin/crawler` | — | Job list + logs |
| Batch jobs | `/admin/batch` | — | Trigger + history |
| Reports queue | `/admin/reports` | — | Moderation |
| Announcements | `/admin/announcements` | — | — |
| System settings | `/admin/settings` | — | Key-value store |
| Mail templates | `/admin/mail/templates` | — | Preview + edit |
| Analytics dashboard | `/admin/analytics` | — | Charts |

---

## 3. URL Structure

```
/                           Homepage (discover, trending, new chapters)
/comics                     Browse all comics (filter, sort)
/comics/:slug               Comic detail page
/comics/:slug/:chapter      Chapter reader
/authors                    Browse authors
/authors/:id                Author detail + comic list
/artists/:id                Artist detail + comic list
/tags                       Tag cloud / browse
/tags/:slug                 Comics with this tag
/groups/:id                 Scanlation group page
/search                     Global search results
/forum                      Forum index
/forum/:slug                Forum board
/forum/:slug/:threadId      Thread + posts

/library                    My shelf
/history                    Reading history
/lists                      My custom lists
/lists/:id                  Custom list detail
/feed                       Activity feed
/notifications              Notifications inbox

/auth/login                 Login page
/auth/register              Registration page
/auth/forgot-password       Forgot password form
/auth/oauth/:provider       OAuth initiation (redirect)

/settings                   Profile settings
/settings/security          Password + sessions
/settings/email             Email preferences

/admin/*                    Admin panel (role: admin/mod)
```

---

## 4. API Client Pattern

### Centralized API client

```typescript
// lib/api/client.ts
const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "https://api.yomira.app/api/v1";

class APIClient {
    private accessToken: string | null = null;

    async fetch<T>(path: string, options?: RequestInit): Promise<T> {
        const res = await fetch(`${BASE_URL}${path}`, {
            ...options,
            headers: {
                "Content-Type": "application/json",
                ...(this.accessToken ? { Authorization: `Bearer ${this.accessToken}` } : {}),
                ...options?.headers,
            },
            credentials: "include",   // send refresh_token cookie
        });

        if (res.status === 401) {
            // Try refresh
            const refreshed = await this.refresh();
            if (refreshed) {
                return this.fetch(path, options);   // retry once
            }
            throw new APIError("UNAUTHORIZED", 401);
        }

        if (!res.ok) {
            const body = await res.json();
            throw new APIError(body.code, res.status, body.error);
        }

        return res.json();
    }

    async refresh(): Promise<boolean> {
        const res = await fetch(`${BASE_URL}/auth/refresh`, {
            method: "POST",
            credentials: "include",
        });
        if (!res.ok) {
            this.accessToken = null;
            return false;
        }
        const { data } = await res.json();
        this.accessToken = data.access_token;
        return true;
    }
}

export const api = new APIClient();
```

### Domain-specific clients

```typescript
// lib/api/comics.ts
export const comicsAPI = {
    list: (params: ComicListParams) =>
        api.fetch<Paginated<Comic>>(`/comics?${toQueryString(params)}`),

    getBySlug: (slug: string) =>
        api.fetch<{ data: ComicDetail }>(`/comics/${slug}`),

    chapters: (id: string, page = 1) =>
        api.fetch<Paginated<Chapter>>(`/comics/${id}/chapters?page=${page}`),

    rate: (id: string, score: number) =>
        api.fetch(`/comics/${id}/rating`, { method: "PUT", body: JSON.stringify({ score }) }),
};
```

---

## 5. Authentication Flow (Client)

### Token storage strategy

- **Access token:** In-memory only (JavaScript variable) — never `localStorage` (XSS risk)
- **Refresh token:** HttpOnly cookie set by server — inaccessible to JavaScript

```typescript
// auth/store.ts — in-memory token store
let _accessToken: string | null = null;

export function setToken(token: string) { _accessToken = token; }
export function getToken(): string | null { return _accessToken; }
export function clearToken() { _accessToken = null; }
```

### Login flow

```
1. User submits login form
2. POST /auth/login → receives { access_token } in response body
3. Store access_token in memory
4. Server sets refresh_token as HttpOnly cookie
5. Redirect to /library
```

### Token refresh on page load

```typescript
// app/layout.tsx — runs on every app load
export async function initAuth() {
    try {
        const { data } = await fetch("/api/v1/auth/refresh", {
            method: "POST",
            credentials: "include",
        }).then(r => r.json());
        setToken(data.access_token);
    } catch {
        clearToken();  // not logged in — fine
    }
}
```

### Logout

```typescript
async function logout() {
    await api.fetch("/auth/logout", { method: "POST" });
    clearToken();
    router.push("/auth/login");
}
```

---

## 6. Reader Modes

The manga reader is the most complex UI component.

### Reading modes

| Mode | Description | Trigger |
|---|---|---|
| `ltr` | Left-to-right pages (Western comics) | User preference |
| `rtl` | Right-to-left pages (manga) | User preference |
| `vertical` | Scroll vertically between pages | User preference |
| `webtoon` | Long strip, vertical scroll, no page gaps | Auto-selected for `format = 'webtoon'` |

### Page fit options

| Fit | Description |
|---|---|
| `width` | Scale page to container width |
| `height` | Scale page to container height |
| `original` | No scaling |
| `stretch` | Fill entire viewport |

### Component structure

```
<Reader>
  <ReaderHeader>       ← chapter title, prev/next nav, settings toggle
  <ReaderCanvas>       ← the actual page display
    <PageImage>        ← lazy-loaded WebP with data saver support
  <ReaderFooter>       ← page number / progress bar
  <ReaderSettings>     ← reading mode, fit, double page, dark mode
```

### Data saver mode

When `readingpreference.datasaver = true`:
- Load low-res pages (`imageurl_sm`) instead of full-res (`imageurl`)
- Preload only 1 page ahead (not 3)

### Keyboard shortcuts

| Key | Action |
|---|---|
| `←` / `A` | Previous page |
| `→` / `D` | Next page |
| `[` | Previous chapter |
| `]` | Next chapter |
| `F` | Toggle fullscreen |
| `S` | Open settings |
| `Esc` | Close settings |

### Auto chapter-read tracking

```typescript
// When user passes 80% of chapter pages → mark as read
useEffect(() => {
    if (currentPage >= Math.floor(totalPages * 0.8)) {
        markChapterRead(chapterId);   // POST /chapters/:id/read
    }
}, [currentPage, totalPages]);
```

---

## 7. Design System Tokens

```css
/* design/tokens.css */

/* Color palette */
--color-primary:       hsl(245 80% 60%);
--color-primary-dark:  hsl(245 80% 45%);
--color-accent:        hsl(340 80% 55%);

/* Dark mode background layers */
--bg-base:             hsl(224 15% 10%);
--bg-surface:          hsl(224 15% 14%);
--bg-elevated:         hsl(224 15% 18%);
--bg-overlay:          hsl(224 15% 22%);

/* Text */
--text-primary:        hsl(0 0% 95%);
--text-secondary:      hsl(0 0% 70%);
--text-muted:          hsl(0 0% 50%);

/* Status */
--color-success:       hsl(142 70% 45%);
--color-warning:       hsl(40 90% 50%);
--color-error:         hsl(0 80% 60%);
--color-info:          hsl(200 80% 55%);

/* Status badge colors (comic status) */
--status-ongoing:      hsl(142 70% 40%);
--status-completed:    hsl(220 80% 60%);
--status-hiatus:       hsl(40 80% 50%);
--status-cancelled:    hsl(0 70% 55%);

/* Typography */
--font-sans:  'Inter', system-ui, sans-serif;
--font-heading: 'Outfit', sans-serif;
--font-mono:  'JetBrains Mono', monospace;

/* Spacing scale (4px base) */
--space-1: 4px;   --space-2: 8px;  --space-3: 12px;
--space-4: 16px;  --space-5: 20px; --space-6: 24px;
--space-8: 32px;  --space-10: 40px; --space-12: 48px;

/* Border radius */
--radius-sm: 4px; --radius-md: 8px; --radius-lg: 12px; --radius-full: 9999px;

/* Shadows */
--shadow-sm:  0 1px 2px hsl(0 0% 0% / 0.3);
--shadow-md:  0 4px 12px hsl(0 0% 0% / 0.4);
--shadow-lg:  0 8px 32px hsl(0 0% 0% / 0.5);
```

---

## 8. State Management

| State | Where | Library |
|---|---|---|
| Auth (access token) | In-memory singleton | Custom store (`auth/store.ts`) |
| User profile | React Query cache | `@tanstack/react-query` |
| Reader preferences | localStorage + API | Synced on login |
| Comic list filters | URL search params | `useSearchParams` |
| Library shelf | React Query cache | `@tanstack/react-query` |
| Notification count | Polling every 60s | React Query `refetchInterval` |

### React Query setup

```typescript
// lib/query.ts
export const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 60_000,       // 1 min
            gcTime: 5 * 60_000,      // 5 min
            retry: 1,
            refetchOnWindowFocus: false,
        },
    },
});
```

---

## 9. Performance Targets

| Metric | Target |
|---|---|
| LCP (Largest Contentful Paint) — comic detail | < 2.5s |
| TBT (Total Blocking Time) | < 200ms |
| CLS (Cumulative Layout Shift) | < 0.1 |
| Page image load (manga page) | < 500ms on 4G |
| Reader first page visible | < 1.5s |
| Search autocomplete response | < 200ms |
| API calls from client | < 300ms p95 |

### Image optimization

- All manga pages stored as **WebP** (converted by Go upload pipeline)
- Covers: `300 × 430` thumbnail + `840 × 1200` HD
- Pages: original resolution WebP
- All images served via Cloudflare CDN with `Cache-Control: public, max-age=31536000`
- Lazy loading with `loading="lazy"` or `IntersectionObserver` in reader
