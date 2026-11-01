# Future CMS Improvements

These are two features discussed but postponed for later implementation.

## 1. Search in Manage Homepage Page Selector

### Current situation
- The Manage Homepage modal uses a native dropdown (`<select>`).
- It displays all public pages.
- There is currently no search input.

### Future improvement
- Add a search field to find pages by title.
- Filter the page selector as the administrator types.
- Keep the existing homepage selection and update functionality unchanged.

**Scope:** Admin UI improvement only. No new API endpoint is required initially, since public pages are already fetched for the selector.

---

## 2. TinyMCE Image Upload and Media Library

### Goal

Allow administrators to upload images directly from their computer through TinyMCE, store them in a shared media location, and reuse the same image across multiple pages.

Page content will continue storing image URLs in its HTML. No dedicated image fields are needed in the Page database model.

### Planned API endpoints

**Base path:** `/api/v1/admin/media`

| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/api/v1/admin/media` | Upload a new image |
| `GET` | `/api/v1/admin/media` | Retrieve media library items |
| `DELETE` | `/api/v1/admin/media/:id` | Delete a media item |

### A. Upload image

`POST /api/v1/admin/media`

- Accept an image file using `multipart/form-data`.
- Validate file type and size.
- Generate a unique filename.
- Store the image in shared media storage.
- Create a media database record containing metadata such as filename, URL, MIME type, and upload date.
- Return the uploaded image URL.

TinyMCE expects a response containing a `location` property, for example:

```json
{
  "location": "/uploads/media/example-image.jpg"
}
```

### B. List media

`GET /api/v1/admin/media`

- Return uploaded images.
- Support pagination.
- Support search by filename.
- Return image metadata and URLs.
- Power the future media library interface.

Possible query parameters:

```text
?page=1&limit=20&search=logo
```

### C. Delete media

`DELETE /api/v1/admin/media/:id`

- Delete the selected media item.
- Before deletion, check whether its URL is referenced in any page content.
- If it is being used, prevent deletion and return a useful error message.
- If unused, delete the file and its media database record.

### TinyMCE integration

- Configure TinyMCE using `images_upload_url` or `images_upload_handler`.
- Connect image uploads to `POST /api/v1/admin/media`.
- Ensure `editor.uploadImages()` runs before saving page content, so Base64 image data is not accidentally stored in the page HTML.

### Planned media library UI

The future media library should allow administrators to:

- Browse uploaded images.
- Search images.
- Preview images.
- Select an existing image for reuse.
- Upload new images.
- Delete unused images.

---

## Suggested Implementation Order

1. Build the backend `POST /api/v1/admin/media` endpoint.
2. Connect TinyMCE image uploads to that endpoint.
3. Ensure image URLs are saved correctly in page HTML.
4. Build `GET /api/v1/admin/media`.
5. Build the media library UI.
6. Implement safe deletion through `DELETE /api/v1/admin/media/:id`.
7. Add search to the Manage Homepage selector.

## Pending Decision

Before implementing the media endpoints, confirm whether the backend API is a separate server or implemented through Next.js route handlers.