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