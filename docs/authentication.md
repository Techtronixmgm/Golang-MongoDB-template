# Authentication & Refresh Token Overview

This document describes the authentication flow used by the boilerplate application.

The authentication system uses:

- JWT access tokens
- JWT refresh tokens
- HTTP-only cookies
- MongoDB for refresh-token session storage
- Password hashing with bcrypt
- Role information inside the access token
- Multiple refresh-token sessions per user

---

## 1. Authentication Architecture

The application uses two types of tokens:

| Token | Purpose | Lifetime | Storage |
|---|---|---|---|
| Access Token | Authenticate API requests | Short-lived | HTTP-only cookie |
| Refresh Token | Obtain a new access token | Long-lived | HTTP-only cookie + hashed DB record |

The access token is used frequently.

The refresh token is used only when a new access token is required.

```text
Client
  │
  ├── Access Token
  │      │
  │      └── Authenticate API requests
  │
  └── Refresh Token
         │
         └── Obtain new Access Token
```

---

# 2. Login Flow

The login endpoint authenticates the user using their login identifier and password.

```text
POST /api/v1/auth/login
        │
        ▼
Auth Handler
        │
        ▼
Auth Service
        │
        ├── Find user
        ├── Check account status
        ├── Compare password with bcrypt
        │
        ▼
Generate Access Token
        │
        ▼
Generate Refresh Token
        │
        ▼
Hash Refresh Token
        │
        ▼
Store hashed token in refresh_tokens
        │
        ▼
Return tokens
        │
        ▼
Set HTTP-only cookies
```

The raw refresh token is **never stored in MongoDB**.

Only its SHA-256 hash is stored.

---

# 3. Access Token

The access token is a JWT containing information such as:

```text
User ID
Role
Issued At
Expires At
```

The token is signed using the application's JWT secret.

The access token is intentionally short-lived.

For example:

```text
Login
  ↓
Access token valid for 2 minutes
  ↓
Expires
```

In a normal deployment, the configured lifetime can be longer.

The important principle is that the access token should have a relatively short lifetime compared with the refresh token.

---

# 4. Refresh Token

The refresh token is also a JWT.

It contains information identifying the user and that it is a refresh token.

Example conceptually:

```text
User ID
Token Type = refresh
Issued At
Expires At
```

The refresh token is also sent using an HTTP-only cookie.

The raw token is not stored in MongoDB.

Instead:

```text
Refresh Token
      │
      ▼
SHA-256 Hash
      │
      ▼
MongoDB
```

---

# 5. Refresh Token Collection

Refresh tokens are stored separately from the `users` collection.

Collection:

```text
refresh_tokens
```

Document structure:

```json
{
  "_id": "ObjectId",
  "user_id": "ObjectId",
  "token_hash": "SHA-256 hash",
  "created_at": "timestamp",
  "last_used_at": "timestamp",
  "expires_at": "timestamp",
  "revoked_at": null
}
```

### Field meanings

### `user_id`

Identifies the user who owns the session.

### `token_hash`

SHA-256 hash of the refresh token.

The raw refresh token is never stored.

### `created_at`

When the refresh-token session was created.

### `last_used_at`

The last time the refresh token was successfully used to obtain an access token.

This is also updated when the application automatically refreshes the access token.

### `expires_at`

The hard expiration time of the refresh token.

### `revoked_at`

Indicates that the refresh-token session has been revoked.

A `null` value means the token has not been revoked.

---

# 6. Why Refresh Tokens Are Stored Separately

The refresh token used to be stored directly on the user:

```text
users
 └── refresh_token_hash
```

That approach allows only one active refresh token per user.

For example:

```text
Login on Device A
      ↓
Refresh Token A stored

Login on Device B
      ↓
Refresh Token B replaces Token A
```

Device A would then lose its refresh session.

The new design stores refresh tokens separately:

```text
users

refresh_tokens
 ├── Token A → User 1
 ├── Token B → User 1
 └── Token C → User 1
```

This allows the same user to have multiple independent sessions.

---

# 7. Multiple Sessions

A user can log in from multiple browsers or devices.

For example:

```text
User
 │
 ├── Browser A
 │      └── Refresh Token A
 │
 ├── Browser B
 │      └── Refresh Token B
 │
 └── Mobile App
        └── Refresh Token C
```

Each session has its own refresh-token document.

Logging out one session only revokes that session's refresh token.

The other sessions remain active.

---

# 8. Refresh Flow

The refresh endpoint can be called directly, but the application also performs automatic refresh through authentication middleware.

The flow is:

```text
API Request
    │
    ▼
Access Token
    │
    ├── Valid ──────────────► Continue request
    │
    └── Expired
          │
          ▼
      Refresh Cookie
          │
          ▼
      Validate JWT
          │
          ▼
      Hash Refresh Token
          │
          ▼
      Find active token in MongoDB
          │
          ▼
      Check User Status
          │
          ▼
      Generate new Access Token
          │
          ▼
      Update last_used_at
          │
          ▼
      Set new Access Cookie
          │
          ▼
      Continue original request
```

The refresh token itself is not replaced during normal refresh.

This is intentional.

---

# 9. Why the Refresh Token Is Not Rotated

The current design does not rotate the refresh token every time it is used.

For example:

```text
Refresh Token A
      │
      ├── Refresh request 1
      │       └── New Access Token
      │
      ├── Refresh request 2
      │       └── New Access Token
      │
      └── Refresh request 3
              └── New Access Token
```

The same refresh token remains valid until:

- It expires
- It is revoked
- The user's session is otherwise invalidated

This avoids a common concurrency problem.

For example, two browser requests can discover an expired access token at nearly the same time:

```text
Request A ──┐
            ├── Refresh Token A
Request B ──┘
```

Both requests can successfully refresh without one invalidating the other.

This is particularly useful when multiple API requests happen simultaneously.

---

# 10. Automatic Refresh

The authentication middleware handles expired access tokens.

A normal request looks like:

```text
Access Token valid
       │
       ▼
Request continues
```

When the access token expires:

```text
Access Token expired
       │
       ▼
Middleware uses Refresh Token
       │
       ▼
New Access Token
       │
       ▼
Request continues
```

The client therefore does not necessarily see a `401` merely because the access token expired.

If the refresh token is also invalid, expired, revoked, or unavailable, authentication fails.

---

# 11. `last_used_at`

`last_used_at` represents the last successful use of the refresh token.

For example:

```text
created_at:    10:00
last_used_at:  10:00

Automatic refresh at 10:15
last_used_at:  10:15

Automatic refresh at 10:30
last_used_at:  10:30
```

It is updated for both:

- Explicit calls to the refresh endpoint
- Automatic refresh performed by the authentication middleware

It does not represent the last API request made by the user.

It specifically represents the last successful refresh-token usage.

---

# 12. Logout Flow

Logout revokes the refresh-token session associated with the current refresh token.

```text
POST /api/v1/auth/logout
        │
        ▼
Validate Refresh Token
        │
        ▼
Find Refresh Token Record
        │
        ▼
Set revoked_at
        │
        ▼
Clear Authentication Cookies
```

After logout:

```text
revoked_at != null
```

The refresh token can no longer be used to obtain a new access token.

---

# 13. Multiple-Session Logout Behavior

Suppose a user has:

```text
Token A → Browser A
Token B → Browser B
Token C → Mobile
```

If Browser A logs out:

```text
Token A → Revoked
Token B → Active
Token C → Active
```

Therefore, logout affects the current refresh-token session rather than every session belonging to the user.

---

# 14. User Status Check

A refresh token by itself does not guarantee that the user can continue using the application.

During refresh, the user is loaded from MongoDB and their account status is checked.

```text
Refresh Token
      │
      ▼
Find User
      │
      ├── Active ──────► Continue
      │
      └── Disabled ────► Reject refresh
```

This allows an administrator to disable a user and prevent that user's refresh token from creating new access tokens.

---

# 15. Refresh Token Expiration

Refresh tokens have their own expiration time.

```text
Access Token
    ↓
Short lifetime

Refresh Token
    ↓
Longer lifetime
```

Once the refresh token expires, it can no longer be used.

MongoDB also has a TTL index on:

```text
expires_at
```

This allows MongoDB to automatically remove expired refresh-token documents.

The application still checks expiration during token lookup; the TTL index is primarily for database cleanup.

---

# 16. MongoDB Indexes

The `refresh_tokens` collection has indexes for the actual access patterns.

### Token hash

```text
token_hash
```

Used to quickly find a refresh-token session.

It is unique because each generated refresh token should correspond to one stored session.

### User ID

```text
user_id
```

Used for operations involving all sessions belonging to a user.

### Expiration

```text
expires_at
```

Configured as a TTL index for automatic cleanup.

---

# 17. Security Properties

The authentication design follows these principles:

### Passwords

Passwords are stored as bcrypt hashes.

```text
Password
   ↓
bcrypt
   ↓
PasswordHash
```

The original password is never stored.

### Refresh tokens

Raw refresh tokens are not stored in MongoDB.

```text
Refresh Token
   ↓
SHA-256
   ↓
token_hash
```

### Cookies

Authentication tokens are transmitted using HTTP-only cookies.

This prevents normal JavaScript access to the cookies.

### Access token

Access tokens are short-lived.

### Refresh token

Refresh tokens are longer-lived and can be individually revoked.

---

# 18. Authentication Endpoints

The main authentication endpoints are:

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
PATCH /api/v1/auth/password
GET /api/v1/auth/me
PATCH /api/v1/auth/me
PATCH /api/v1/auth/me/image
```

The refresh endpoint does not require a valid access token because its purpose is to obtain a new access token when the existing one is expired.

---

# 19. Important Design Decisions

The following decisions are intentional parts of this boilerplate.

### Refresh tokens are stored separately

This allows multiple sessions per user.

### Raw refresh tokens are never stored

Only their hashes are stored.

### Refresh tokens are not rotated on every refresh

This avoids concurrent-refresh race conditions and keeps the implementation simple.

### Refresh-token sessions are individually revocable

Logging out one device does not automatically log out every other device.

### Access and refresh lifetimes are independent

The access token is short-lived while the refresh token has a longer lifetime.

### Automatic refresh is handled by middleware

The application can transparently renew an expired access token when a valid refresh session exists.

---

# 20. Basic Authentication Test Checklist

After authentication changes, verify the following:

- [ ] Login succeeds with valid credentials
- [ ] Invalid credentials are rejected
- [ ] Access cookie is created
- [ ] Refresh cookie is created
- [ ] Refresh-token document is created
- [ ] Raw refresh token is not stored in MongoDB
- [ ] Access token expires as configured
- [ ] Automatic refresh generates a new access token
- [ ] `last_used_at` is updated after refresh
- [ ] Direct `/auth/refresh` works
- [ ] Concurrent refresh requests both work
- [ ] Logout revokes the current refresh session
- [ ] Revoked refresh token cannot be used again
- [ ] Multiple sessions can exist for the same user
- [ ] Logging out one session does not revoke another session
- [ ] Disabled users cannot refresh their session
- [ ] Expired refresh tokens are rejected
- [ ] Expired refresh-token documents are eventually removed by MongoDB TTL

---

# 21. Overall Flow

The complete authentication lifecycle can be summarized as:

```text
                         LOGIN
                           │
                           ▼
                    Verify Credentials
                           │
                           ▼
                  ┌──────────────────┐
                  │                  │
                  ▼                  ▼
             Access Token       Refresh Token
                  │                  │
                  │                  ▼
                  │            Hash + Store
                  │                  │
                  ▼                  ▼
             HTTP Cookie       HTTP Cookie
                  │
                  ▼
             API Requests
                  │
                  ▼
          Access Token Valid?
             │           │
            Yes          No
             │           │
             │           ▼
             │      Refresh Token
             │           │
             │           ▼
             │      Validate + Lookup
             │           │
             │           ▼
             │      Generate Access
             │           │
             │           ▼
             │      Update last_used_at
             │           │
             └───────────┤
                         ▼
                  Continue Request
                         │
                         ▼
                       LOGOUT
                         │
                         ▼
                  Revoke Session
                         │
                         ▼
                  Clear Cookies
```

This design provides a simple session-based refresh-token system while remaining lightweight enough for a reusable backend boilerplate.