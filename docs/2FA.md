# Two-Factor Authentication (2FA)

## Overview

The application supports user-level TOTP-based two-factor authentication (2FA), with a global application setting that controls whether configured 2FA is enforced during login.

The implementation uses:

- TOTP according to RFC 6238
- `github.com/pquerna/otp`
- AES-256-GCM for encrypting stored TOTP secrets
- HTTP-only authentication cookies
- Short-lived 2FA login challenges
- Backup codes as an additional login method

## 2FA Configuration

Relevant configuration:

- `TOTP_ENCRYPTION_KEY` — server-side secret used to derive the AES-256 key.
- `BACKUP_CODE_COUNT` — number of backup codes generated for a user.
- `BACKUP_CODE_LOW_THRESHOLD` — threshold at which the frontend is informed that backup codes are running low.

Default backup-code configuration is 10 codes with a low threshold of 4.

When global 2FA is enabled, startup validates the TOTP encryption key and backup-code configuration. If the configuration is invalid, the application does not start.

If global 2FA is disabled, these 2FA configuration checks do not prevent startup.

## TOTP Settings

The TOTP service uses:

- Period: 30 seconds
- Digits: 6
- Algorithm: SHA-1
- Secret size: 20 bytes
- Allowed clock skew: 1 period
- Time zone: UTC

The user's username is currently used as the TOTP account name.

## TOTP Secret Storage

TOTP secrets are never stored as plaintext in MongoDB.

The configured `TOTP_ENCRYPTION_KEY` is hashed with SHA-256 to obtain a 32-byte AES-256 key. The TOTP secret is then encrypted using AES-256-GCM with a random nonce.

Stored value:

```text
TOTP secret
    ↓
AES-256-GCM encryption
    ↓
Base64 ciphertext
    ↓
MongoDB
```

The TOTP secret fields are excluded from JSON responses.

The `otpauthUrl` returned during setup contains the secret and must not be logged or exposed unnecessarily.

## User 2FA Fields

The user record contains:

- `TwoFactorEnabled`
- `TwoFactorSecret`
- `TwoFactorPendingSecret`
- `BackupCodeHashes`

The actual TOTP secret and backup-code hashes are never returned in API responses.

## Setup Flow

### 1. Start Setup

`POST /api/v1/auth/2fa/setup`

Authentication is required.

The server:

1. Loads the current user.
2. Verifies that the account is active.
3. Verifies that 2FA is not already enabled.
4. Generates a TOTP secret.
5. Encrypts the secret.
6. Stores it as the pending secret.
7. Returns the secret and `otpauthUrl`.

The frontend can use the returned `otpauthUrl` directly with its QR-code library.

### 2. Verify Setup

`POST /api/v1/auth/2fa/verify-setup`

Authentication is required.

The client sends the current six-digit TOTP code.

After successful verification:

- The pending secret becomes the active encrypted secret.
- `TwoFactorEnabled` becomes `true`.
- The pending secret is cleared.
- Backup-code hashes are stored.
- The plaintext backup codes are returned once.

The user must save the backup codes because they are not shown again.

## Login Flow

When global 2FA enforcement is enabled and the user has 2FA enabled:

```text
Password login
    ↓
2FA challenge
    ↓
TOTP or backup code
    ↓
Access + refresh tokens
```

The initial password login does not issue normal access/refresh tokens in this case.

Instead, it issues a short-lived `2fa_challenge` JWT with a five-minute lifetime.

The normal authentication middleware rejects this challenge token.

The 2FA verification endpoint is intentionally outside the normal access-token middleware.

### Verify 2FA Login

`POST /api/v1/auth/2fa/verify-login`

The client sends:

- `challengeToken`
- `code`

A valid TOTP code completes login.

A valid backup code also completes login and consumes that backup code.

The response includes:

- login success message
- user information
- remaining backup-code count
- low-backup-code status

## Backup-Code Policy

Backup codes are an emergency recovery mechanism.

They can be used during login when the user cannot provide a TOTP code.

They cannot be used to:

- Disable 2FA
- Regenerate backup codes
- Change the authenticator/2FA configuration

Those operations require the current TOTP code.

This keeps backup codes as a recovery path rather than a general replacement for the authenticator.

## Disable 2FA

`POST /api/v1/auth/2fa/disable`

Authentication is required.

A valid current TOTP code is required.

After successful disable:

- `TwoFactorEnabled` becomes `false`
- Active TOTP secret is cleared
- Pending TOTP secret is cleared
- Backup-code hashes are cleared

## Global 2FA

Global 2FA is an application-level enforcement setting.

### Global OFF

If global 2FA is disabled:

- Users with 2FA enabled can log in normally.
- Their personal 2FA configuration remains stored.
- User 2FA is not deleted or reset.

### Global ON

If global 2FA is enabled:

- Users who have 2FA enabled must complete 2FA during login.
- Users who have not enabled personal 2FA can still log in normally.

Global 2FA therefore acts as an enforcement switch; it does not automatically enable 2FA for every user.

## Settings API

Public settings expose only settings intended for frontend/public use.

Administrative settings expose the application's 2FA state to authorized administrators.

Global 2FA can be enabled or disabled through the admin settings flow.

Enabling global 2FA requires a valid `TOTP_ENCRYPTION_KEY`.

Disabling global 2FA does not modify individual users' 2FA data.

## Startup Safety

On application startup:

1. MongoDB is connected.
2. Application settings are initialized if necessary.
3. The persisted global 2FA setting is checked.
4. If global 2FA is enabled, the required 2FA configuration is validated.
5. The HTTP server starts only after successful validation.

The application does not silently disable global 2FA when configuration is invalid.

## Tested Behavior

The following 2FA behavior has been tested:

- TOTP setup and verification
- Encrypted TOTP secret storage
- TOTP login verification
- Backup-code login
- Consumption of used backup codes
- Rejection of invalid and previously used backup codes
- TOTP login without consuming backup codes
- Low backup-code threshold behavior
- TOTP-only 2FA disable
- Global 2FA enable/disable behavior
- Startup failure with invalid 2FA configuration

## Security Notes

- TOTP secrets are encrypted at rest.
- TOTP and backup-code secrets are excluded from API responses where appropriate.
- Backup codes are stored as hashes rather than plaintext.
- 2FA login challenges are short-lived.
- The challenge token cannot be used as a normal access token.
- The encryption key must be supplied through server configuration and must not be committed to source control.
- `otpauthUrl` and TOTP secrets must not be logged.
