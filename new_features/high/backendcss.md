Here is a brief, architectural reference for implementing database-backed custom CSS in your Go (Gin) + Next.js stack whenever you decide to add it.

---

### Core Concept

Instead of writing CSS to local disk files (which get overwritten on deployment and cannot be dynamically recompiled in production), store raw CSS as a text field in your database. Next.js fetches this string at runtime and serves it as an inline `<style>` block in the public HTML `<head>`.

```
[ Admin Dashboard: Code Editor ]
            │
            ▼ PUT /api/v1/admin/settings (custom_css)
    [ Gin API + Database ]
            │
            ▼ GET /api/v1/settings (or fetched directly in Server Layout)
 [ Next.js Root / Public Layout ]
            │
            ▼
    <head>
      <style id="site-custom-css">/* Stored rules */</style>
    </head>

```

---

### 1. Database Schema

Add a column or document field to your existing site configuration/settings table:

* **SQL (PostgreSQL / MySQL):** `custom_css TEXT` (or `LONGTEXT`)
* **MongoDB / NoSQL:** `custom_css: { type: String, default: "" }`

---

### 2. Gin API Layer

Two endpoints manage the data:

* **Admin Route (`PUT /api/v1/admin/settings`):**
* Protected by your JWT/Session `admin` middleware.
* Accepts a JSON payload like `{"custom_css": "..."}`.
* Strips any dangerous HTML tags (e.g., `<script>`, `</style>`) before saving so only raw CSS text reaches the database.


* **Public Route (`GET /api/v1/settings`):**
* Open endpoint returning site metadata, brand settings, and `custom_css`.
* Publicly readable with standard caching (or no-store/revalidate tags).



---

### 3. Next.js Delivery (`app/layout.tsx` or Page Wrapper)

In your async Server Component layout:

```tsx
// Fetched on the server during the incoming request
const settings = await getSettings();

return (
  <html lang="en">
    <head>
      {settings?.custom_css && (
        <style
          id="custom-site-css"
          dangerouslySetInnerHTML={{ __html: settings.custom_css }}
        />
      )}
    </head>
    <body>{children}</body>
  </html>
);

```

---

### Why This Architecture Works

* **Zero Redeploys:** Changes take effect immediately upon saving—no `npm run build` or container restart required.
* **Cloud-Safe:** Fully compatible with containerized, multi-instance, or serverless deployments (Docker, AWS, Railway) where local file systems are read-only or ephemeral.
* **No Cache Staling:** Because the rules are injected server-side on render, visitors always receive the latest styles without dealing with browser cache invalidation queries (`?v=1.2`).