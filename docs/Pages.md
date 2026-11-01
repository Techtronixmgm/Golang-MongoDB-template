## Pages

Pages are used to store and manage website content.

User can create, update and delete pages.

### Page Information

- **Title:** Page title.
- **Content:** Main page content.
- **Slug:** Unique, URL-friendly identifier for the page.
- **Visibility:** Controls access to the page (`public` or `registered`).
- **CreatedAt:** Date and time the page was created.
- **UpdatedAt:** Date and time the page was last updated.

### Page Indexes

The following MongoDB indexes are used by the Pages collection:

| Index | Purpose |
|---|---|
| `slug` (unique) | Ensures every page has a unique slug. |
| `visibility` + `created_at` | Supports filtering pages by visibility and sorting by creation date. |
| `author_id` + `created_at` | Supports retrieving pages by author and sorting by creation date. |
| `is_homepage` (unique, partial) | Ensures that only one page can have `is_homepage: true`. |

The `is_homepage` index is a unique partial index. It applies only to documents where `is_homepage` is `true`, allowing multiple pages to have `false` or no value.

### Homepage Feature

- A page can be designated as the website homepage.
- Only public pages can be assigned as the homepage.
- Only one page can be the homepage at a time.
- Assigning a new homepage automatically removes the designation from the previous one.
- Homepage designation is optional; there may be no homepage.
- The current homepage cannot be deleted.
- Changing the current homepage's visibility to `registered` is rejected.
- Homepage designation is managed after page creation, not during creation.

### Concurrent Homepage Assignment

- A unique partial index on `is_homepage` ensures that at most one page can have `is_homepage: true`.
- If two different pages are assigned as homepage simultaneously, one operation may fail with a duplicate-key error.
- Atomic transaction-based handling and automated concurrency testing are deferred.
