# Menu Features

The backend provides a configurable menu system for managing navigation menus from the admin side.

## Menu Locations

Three menu locations are supported:

- **Top** — Main/top navigation
- **Left** — Sidebar navigation
- **Bottom** — Footer navigation

Only one menu can exist for each location.

## Menu Management

Administrators can:

- Create a menu
- View a list of menus
- View an individual menu
- Update the menu name and status
- Delete a menu

A menu has:

- Name
- Location
- Status
- Menu items
- Created and updated timestamps

## Menu Items

Menu items support three types:

- **Page** — Links to an existing backend page
- **URL** — Uses a custom URL
- **Group** — Acts as a container for child items

Each item has:

- Label
- Type
- Order
- Display status
- Optional page reference or URL
- Optional child items

## Nested Menus

Menu groups can contain child items.

The system supports nested menu structures with a configurable maximum depth for each menu location.

For example:

```text
Top
├── Item
├── Group
│   ├── Item
│   └── Group
│       └── Item
```

The maximum allowed depth can differ between the `top`, `left`, and `bottom` menus.

## Menu Item Management

Administrators can:

- Add menu items
- Edit menu items
- Delete menu items
- Change item display status
- Change item type
- Move an item within its current sibling level

When an item is deleted, the remaining sibling items are automatically resequenced.

Deleting a group also removes its nested children because menu items are stored within the menu structure.

## Menu Item Validation

The backend validates:

- Menu item type
- Required page references
- Referenced pages
- Required URLs
- Group restrictions
- Parent item existence
- Parent item type
- Maximum nesting depth

A group can contain children, while page and URL items cannot.

## Public Menus

Public menu data can be retrieved by menu location.

Only active menus and visible menu items are included in the public menu response.

For page-based items, the current page slug is used to construct the public page URL, so changing a page slug does not require updating the menu item.

## Authorization

Menu management is restricted to administrators.

Public menu retrieval does not require administrator access.

## Design

Menu items are embedded within the menu document. This keeps menu management simple and allows an entire menu hierarchy to be managed as a single structure.

# Menu Settings

Menu nesting depth can be configured separately for each menu location through Application Settings.

## Maximum Depth

| Location | Default |
|---|---:|
| Top | 3 |
| Left | 2 |
| Bottom | 2 |

Example:

```json
{
  "menu_max_depth": {
    "top": 3,
    "left": 2,
    "bottom": 2
  }
}
```

### Behavior

The configured depth controls how deeply new menu items can be nested.

Changing the maximum depth does **not** modify or remove existing menu items.

For example, if the Top menu currently allows 3 levels and is changed to 2, existing level-3 items remain unchanged. New items cannot be added beyond level 2.

A minimum depth of `1` is required for each menu location.

# Public Menu API Optimization

## Overview
The existing public menu API was optimized to resolve page-linked menu items and group destinations efficiently. No new API endpoint was introduced.

---

## Changes Made

| Aspect | Before Optimization | After Optimization |
| :--- | :--- | :--- |
| **API endpoint** | Existing public API | Same public API; no new endpoint |
| **Menu retrieval** | One database query | One database query |
| **Page retrieval** | Individual database query for each page-linked item | One batch query for all unique referenced pages |
| **Nested groups** | Individual page lookups where applicable | Reuses the same fetched page map |
| **Page URL resolution** | Fetches page details individually to resolve the slug | Resolves URLs using batch-fetched page details |
| **Group destinations** | Page-linked group URLs were not resolved in the public response | Supports groups linked to a page or custom URL while retaining children |
| **Custom URLs** | Preserved | Preserved |
| **Response structure** | Existing public response contract | Existing response structure maintained |
| **Performance** | Database queries increase with the number of page-linked items | Fewer database queries and reduced database overhead |

---

## Implementation Details

* **Added batch fetching:** Introduced `FindByIDs()` in the page repository to retrieve multiple pages using a single database query.
* **Collected page IDs:** Updated `GetPublicByLocation()` to collect unique page IDs recursively from all menu levels.
* **Reused fetched page data:** Updated `buildPublicMenuItems()` to resolve page URLs using the batch-fetched page map instead of querying the database for each item.
* **Supported group destinations:** Ensured groups can have a page destination, a custom URL, or no destination while retaining their nested children.

---

## Validation

Executed the following command:

```bash
go test ./...
```

All packages compiled successfully. However, the output reported no test files, so runtime behavior and database query counts still require verification.

## Conclusion

The existing public menu API was optimized through batch page fetching and in-memory page lookup. This reduces database round trips while preserving the existing endpoint and response structure. No new endpoint was created.

