# Team members and invites

Postchi workspaces are multi-user. Teammates can register on their own, then workspace owners add them directly or share an invite link.

## Roles

| Role | Read | Edit | Execute | Delete workspace | Manage members |
|------|:----:|:----:|:-------:|:----------------:|:--------------:|
| **viewer** | Yes | No | Yes | No | No |
| **editor** | Yes | Yes | Yes | No | No |
| **owner** | Yes | Yes | Yes | Yes | Yes |

Owners manage teammates under **Workspace settings → Team**.

## Adding teammates

Enter an email and role, then click **Add teammate**.

| Situation | What happens |
|-----------|----------------|
| Email already registered | User is added to the workspace immediately (`POST /api/workspaces/:id/invites` returns `outcome: "added"`) |
| Email not registered | A pending invite is created with a copyable link (`outcome: "invited"`, `invite_url`) |

### Invite links without email

SMTP is optional. When `SMTP_HOST` is not set, owners copy the invite link from the pending invites list (or it is auto-copied after creation) and share it via Slack, chat, or any channel.

When SMTP is configured, owners can optionally check **Send email invite** to deliver the link by email instead of copying manually.

Invite links expire after `INVITE_TTL_HOURS` (default: 168 hours / 7 days).

### Accepting an invite

Invitees open `/invite/{token}`:

- **New user**: set password (and optional display name), then join the workspace and sign in
- **Existing user**: accept the invite, then sign in with their account

## Self-registration and domain allowlist

Anyone can register at `/register` by default (`POST /api/auth/register`).

To restrict self-signup to your organization, set:

```env
REGISTRATION_ALLOWED_EMAIL_DOMAINS=yourcompany.com,subsidiary.com
```

- Only emails on those domains can register
- Invites are **not** restricted by this setting (owners can still invite any email)
- The register page shows which domains are allowed when the setting is active

Public config (no auth): `GET /api/config/public`

```json
{
  "smtp_configured": true,
  "registration_allowed_domains": ["yourcompany.com"]
}
```

## Change password

Registered users with local accounts can change their password from **Account** on the workspaces list or workspace toolbar.

API: `POST /api/auth/change-password` (authenticated)

```json
{
  "current_password": "existing-secret",
  "new_password": "new-secret"
}
```

On success, other refresh sessions are revoked and a new token pair is returned so the current browser session stays signed in.

## API summary

| Endpoint | Description |
|----------|-------------|
| `GET /api/config/public` | SMTP and registration domain flags |
| `POST /api/auth/change-password` | Change password (local accounts) |
| `POST /api/workspaces/:id/invites` | Add registered user or create invite (`send_email` optional) |
| `GET /api/workspaces/:id/invites` | List pending invites (includes `invite_url`) |
| `DELETE /api/workspaces/:id/invites/:inviteId` | Revoke invite |
| `POST /api/workspaces/:id/members` | Direct add by email (registered users only) |
| `GET /api/invites/:token` | Preview invite (public) |
| `POST /api/invites/:token/accept` | Accept invite (public) |

### Create invite / add teammate body

```json
{
  "email": "colleague@company.com",
  "role": "editor",
  "send_email": true
}
```

`send_email` defaults to `true` when SMTP is configured, otherwise `false`.
