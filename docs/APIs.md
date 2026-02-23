# Client Hub Open Project - API Documentation

Base URL: `/` (Most endpoints use the `/api` prefix)

---

## Permission System

This API uses Role-Based Access Control (RBAC) with granular permissions. Each endpoint requires specific permissions beyond basic authentication.

**Permission Format:** `resource:action` (e.g., `clients:read`, `users:create`)

**Built-in Roles:**

- **root**: Full access to all resources
- **admin**: Administrative access with some restrictions
- **user**: Standard user access for daily operations
- **viewer**: Read-only access

**Permission Checking:** The system checks if the authenticated user has the required permission for the resource and action. Root users bypass permission checks.

---

## Authentication

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `POST` | `/api/login` | Public | `username`, `password` | `token`, `refresh_token`, `user_id`, `username`, `role`, `display_name`, `permissions[]` |
| `POST` | `/api/refresh-token` | Public | `refresh_token` | `token` |
| `POST` | `/api/logout` | Authenticated | N/A | `message` |

---

## Users

*System user management.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/users` | Authenticated with `users:read` permission | N/A | List of users (`id`, `username`, `role`, `display_name`, `failed_attempts`, `lock_level`, `locked_until`) |
| `POST` | `/api/users` | Authenticated with `users:create` permission (role-specific: `users:create_admin` for admin role, `users:create_user` for user, etc.) | `username`, `display_name`, `password`, `role` (user/admin/root) | `id`, `message` |
| `GET` | `/api/users/{username}` | Authenticated with `users:read` permission | N/A | Complete User Object |
| `PUT` | `/api/users/{username}` | Authenticated with `users:update` permission (or self) | `display_name`, `password` (Optional: `username`, `role` - requires `users:manage_roles`) | `message` |
| `PUT` | `/api/users/{username}/block` | Authenticated with `users:block` permission | N/A | `message` |
| `PUT` | `/api/users/{username}/unlock` | Authenticated with `users:block` permission | N/A | `message` |

**Security Rules:**

- Role-specific permissions apply (e.g., `users:create_admin` required to create admin users)
- `users:manage_roles` required to change roles
- Users can update their own data without special permissions
- Root has all permissions implicitly

---

## Clients

*Client/Company registration management.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/clients` | Authenticated with `clients:read` permission | Optional Query: `include_stats=true` | List of Clients (`id`, `name`, `registration_id`, `nickname`, `birth_date`, `email`, `phone`, `address`, `notes`, `status`, `tags`, `contact_preference`, `last_contact_date`, `next_action_date`, `created_at`, `documents`, `archived_at`) + optional stats |
| `GET` | `/api/clients/counts` | Authenticated | N/A | `total`, `active`, `inactive`, `archived` |
| `POST` | `/api/clients` | Authenticated with `clients:create` permission | `name` (Required), `registration_id`*, `nickname`*, `birth_date`*, `email`*, `phone`*, `address`*, `notes`*, `tags`*, `contact_preference`*, `documents`* (*=Optional) | `id`, `message` |
| `GET` | `/api/clients/{id}` | Authenticated with `clients:read` permission | N/A | Complete Client Object |
| `PUT` | `/api/clients/{id}` | Authenticated with `clients:update` permission | `name`, `registration_id`, `nickname`, `email`, `phone`, `address`, `notes`, `tags`, `contact_preference`, `documents` | `message` |
| `DELETE` | `/api/clients/{id}` | Authenticated with `clients:delete` permission | N/A | 204 No Content |
| `PUT` | `/api/clients/{id}/archive` | Authenticated with `clients:archive` permission | N/A | `message` |
| `PUT` | `/api/clients/{id}/unarchive` | Authenticated with `clients:archive` permission | N/A | `message` |
| `GET` | `/api/clients/{id}/affiliates` | Authenticated with `affiliates:read` permission | N/A | List of Affiliates |
| `POST` | `/api/clients/{id}/affiliates` | Authenticated with `affiliates:create` permission | `name` (Required), `description`*, `birth_date`*, `email`*, `phone`*, `address`*, `notes`*, `tags`*, `contact_preference`*, `documents`* | `id`, `message` |

**With `include_stats=true`:** Response includes `active_contracts`, `expired_contracts`, `archived_contracts` per client.

---

## Affiliates

*Branches, departments, or affiliates linked to a Client.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `PUT` | `/api/affiliates/{id}` | Authenticated with `affiliates:update` permission | `name`, `description`, `birth_date`, `email`, `phone`, `address`, `notes`, `tags`, `contact_preference`, `documents` | `message` |
| `DELETE` | `/api/affiliates/{id}` | Authenticated with `affiliates:delete` permission | N/A | `message` |

---

## Contracts

*Contracts signed with Clients.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/contracts` | Authenticated with `contracts:read` permission | N/A | List of Contracts (`id`, `model`, `item_key`, `start_date`, `end_date`, `subcategory_id`, `client_id`, `affiliate_id`, `archived_at`) |
| `POST` | `/api/contracts` | Authenticated with `contracts:create` permission | `client_id`, `subcategory_id`, `model`, `item_key` (Optional: `start_date`, `end_date`, `affiliate_id`) | `id`, `message` |
| `GET` | `/api/contracts/{id}` | Authenticated with `contracts:read` permission | N/A | Complete Contract Object |
| `PUT` | `/api/contracts/{id}` | Authenticated with `contracts:update` permission | `model`, `item_key`, `start_date`, `end_date`, `subcategory_id`, `client_id`, `affiliate_id` | `message` |
| `PUT` | `/api/contracts/{id}/archive` | Authenticated with `contracts:archive` permission | N/A | `message` |
| `GET` | `/api/contracts/{id}/financial` | Authenticated with `financial:read` permission | N/A | Contract Financial Object (or `null` if none) |

**Contract Status Logic:**

- `start_date = null`: Contract always started (infinite lower bound)
- `end_date = null`: Contract never expires (infinite upper bound)
- Calculated Status: `Ativo`, `Expirando em Breve` (≤30 days to end), `Expirado`

---

## Financial

*Financial models and installments for contracts. Supports three financial types: unique (one-time), recurring, and custom (with installments).*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/financial` | Authenticated with `financial:read` permission | Optional Query: `page`, `limit` | List of Financial (`id`, `contract_id`, `client_id`, `client_name`, `contract_model`, `contract_item_key`, `financial_type`, `recurrence_type`, `due_day`, `client_value`, `received_value`, `description`, `is_active`, `total_client_value`, `total_received_value`, `total_installments`, `paid_installments`) with pagination metadata |
| `POST` | `/api/financial` | Authenticated with `financial:create` permission | `contract_id`, `financial_type` (Optional: `recurrence_type`, `due_day`, `client_value`, `received_value`, `description`, `installments[]`) | `id`, `message` |
| `GET` | `/api/financial/{id}` | Authenticated with `financial:read` permission | N/A | Complete Financial Object with Installments |
| `PUT` | `/api/financial/{id}` | Authenticated with `financial:update` permission | `financial_type` (Optional: `recurrence_type`, `due_day`, `client_value`, `received_value`, `description`, `is_active`, `installments[]`) | `message` |
| `DELETE` | `/api/financial/{id}` | Authenticated with `financial:delete` permission | N/A | `message` |

**Financial Types:**

- `unico`: One-time financial (uses `client_value` and `received_value` directly)
- `recorrente`: Recurring financial (requires `recurrence_type`: `mensal`, `trimestral`, `semestral`, `anual`)
- `personalizado`: Custom installments (uses `installments[]` array)

**Values:**

- `client_value`: Amount the client pays
- `received_value`: Amount you receive (commission, etc.)

### Example: Create a custom financial with installments

```bash
curl -X POST http://localhost:3000/api/financial \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "contract_id": "550e8400-e29b-41d4-a716-446655440000",
    "financial_type": "personalizado",
    "description": "3-month payment plan"
  }'
```

Response:

```json
{
  "message": "Financial created successfully",
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001"
  }
}
```

---

## Financial Installments

*Custom installments for financial with type `personalizado`.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/financial/{id}/installments` | Authenticated with `financial:read` permission | N/A | List of Installments |
| `POST` | `/api/financial/{id}/installments` | Authenticated with `financial:create` permission | `installment_number`, `client_value`, `received_value` (Optional: `installment_label`, `due_date`, `notes`) | `id`, `message` |
| `PUT` | `/api/financial/{id}/installments/{inst_id}` | Authenticated with `financial:update` permission | `client_value`, `received_value` (Optional: `installment_label`, `due_date`, `notes`) | `message` |
| `DELETE` | `/api/financial/{id}/installments/{inst_id}` | Authenticated with `financial:delete` permission | N/A | `message` |
| `PUT` | `/api/financial/{id}/installments/{inst_id}/pay` | Authenticated with `financial:mark_paid` permission | N/A | `message` |
| `PUT` | `/api/financial/{id}/installments/{inst_id}/unpay` | Authenticated with `financial:mark_paid` permission | N/A | `message` |

**Installment Status:** `pendente`, `pago`, `atrasado`, `cancelado`

**Installment Labels:** `installment_number` 0 = "Entrada", 1 = "1ª Parcela", etc.

### Example: Create an installment

```bash
curl -X POST http://localhost:3000/api/financial/660e8400-e29b-41d4-a716-446655440001/installments \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "installment_number": 0,
    "client_value": 1000.00,
    "received_value": 800.00,
    "installment_label": "Entrada",
    "due_date": "2025-02-15T00:00:00Z"
  }'
```

Response:

```json
{
  "message": "Installment created successfully",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002"
  }
}
```

### Example: Mark installment as paid

```bash
curl -X PUT http://localhost:3000/api/financial/660e8400-e29b-41d4-a716-446655440001/installments/770e8400-e29b-41d4-a716-446655440002/pay \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json"
```

Response:

```json
{
  "message": "Installment marked as paid"
}
```

---

## Financial Dashboard

*Summary and upcoming financial for dashboard widgets.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/financial/summary` | Authenticated with `financial:read` permission | Optional Query: `year`, `month` | `total_to_receive`, `total_client_pays`, `already_received`, `pending_count`, `paid_count`, `overdue_count` |
| `GET` | `/api/financial/detailed-summary` | Authenticated with `financial:read` permission | N/A | Detailed summary with period breakdown (see below) |
| `GET` | `/api/financial/upcoming` | Authenticated with `financial:read` permission | Optional Query: `days` (default: 30) | List of Upcoming Financial (`installment_id`, `contract_id`, `client_id`, `client_name`, `contract_model`, `installment_label`, `client_value`, `received_value`, `due_date`, `status`) |
| `GET` | `/api/financial/overdue` | Authenticated with `financial:read` permission | N/A | List of Overdue Financial (same fields as upcoming) |

### Example: Get financial summary

```bash
curl -X GET "http://localhost:3000/api/financial/summary?year=2025&month=2" \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "message": "Financial summary retrieved",
  "data": {
    "total_to_receive": 50000.00,
    "total_client_pays": 62500.00,
    "already_received": 35000.00,
    "pending_count": 15,
    "paid_count": 8,
    "overdue_count": 2
  }
}
```

**Detailed Summary Response Fields:**

- `total_to_receive`: Total amount to receive across all contracts
- `total_client_pays`: Total amount clients pay
- `total_received`: Total amount already received
- `total_pending`: Total amount still pending
- `total_overdue`: Total amount overdue
- `total_overdue_count`: Number of overdue installments
- `last_month`: Period summary for last month
- `current_month`: Period summary for current month
- `next_month`: Period summary for next month
- `monthly_breakdown`: Array of period summaries for past 3 months to next 3 months
- `generated_at`: Timestamp when the summary was generated
- `current_date`: Current date

**Period Summary Fields:**

- `period`: Period identifier (e.g., "2025-01")
- `period_label`: Human-readable label (e.g., "Janeiro 2025")
- `total_to_receive`: Total to receive in the period
- `total_client_pays`: Total clients pay in the period
- `already_received`: Amount already received in the period
- `pending_amount`: Amount still pending in the period
- `pending_count`: Number of pending installments
- `paid_count`: Number of paid installments
- `overdue_count`: Number of overdue installments
- `overdue_amount`: Total overdue amount in the period

---

## Categories

*Classification of services or products.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/categories` | Authenticated with `categories:read` permission | Optional Query: `include_archived=true` | List of Categories (`id`, `name`, `status`, `archived_at`) |
| `POST` | `/api/categories` | Authenticated with `categories:create` permission | `name` | `id`, `message` |
| `GET` | `/api/categories/{id}` | Authenticated with `categories:read` permission | N/A | Category Object |
| `PUT` | `/api/categories/{id}` | Authenticated with `categories:update` permission | `name` | `message` |
| `DELETE` | `/api/categories/{id}` | Authenticated with `categories:delete` permission | N/A | `message` |
| `POST` | `/api/categories/{id}/archive` | Authenticated with `categories:archive` permission | N/A | `message` |
| `POST` | `/api/categories/{id}/unarchive` | Authenticated with `categories:archive` permission | N/A | `message` |
| `GET` | `/api/categories/{id}/subcategories` | Authenticated with `subcategories:read` permission | Optional Query: `include_archived=true` | List of Subcategories |

---

## Subcategories

*Subcategories/Lines within Categories.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/subcategories` | Authenticated with `subcategories:read` permission | N/A | List of Subcategories (`id`, `name`, `category_id`, `archived_at`) |
| `POST` | `/api/subcategories` | Authenticated with `subcategories:create` permission | `name`, `category_id` | `id`, `message` |
| `GET` | `/api/subcategories/{id}` | Authenticated with `subcategories:read` permission | N/A | Subcategory Object |
| `PUT` | `/api/subcategories/{id}` | Authenticated with `subcategories:update` permission | `name` | `message` |
| `DELETE` | `/api/subcategories/{id}` | Authenticated with `subcategories:delete` permission | N/A | `message` |
| `POST` | `/api/subcategories/{id}/archive` | Authenticated with `subcategories:archive` permission | N/A | `message` |
| `POST` | `/api/subcategories/{id}/unarchive` | Authenticated with `subcategories:archive` permission | N/A | `message` |

---

## Roles & Permissions

*Granular access control (RBAC).*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/roles` | Authenticated with `roles:read` permission | Optional Query: `include_permissions=true` | List of Roles (`id`, `name`, `display_name`, `description`, `is_system`, `is_active`, `priority`) |
| `POST` | `/api/roles` | Authenticated with `roles:create` permission | `name` (slug), `display_name`, `priority` (Optional: `description`) | `id`, `name` |
| `GET` | `/api/roles/{id}` | Authenticated with `roles:read` permission | Optional Query: `include_permissions=true` | Role Object (optionally with Permissions) |
| `PUT` | `/api/roles/{id}` | Authenticated with `roles:update` permission | `display_name`, `priority` (Optional: `description`) | `id`, `display_name` |
| `DELETE` | `/api/roles/{id}` | Authenticated with `roles:delete` permission | N/A | `message` |
| `GET` | `/api/roles/{id}/permissions` | Authenticated with `roles:read` permission | N/A | List of Permissions for Role |
| `PUT` | `/api/roles/{id}/permissions` | Authenticated with `roles:manage_permissions` permission | `permission_ids` (Array of UUIDs) | `message` |
| `GET` | `/api/permissions` | Authenticated with `roles:read` permission | N/A | List of All Available Permissions (`id`, `resource`, `action`, `display_name`, `description`, `category`) |
| `GET` | `/api/user/permissions` | Authenticated | N/A | `user_id`, `role`, `role_id`, `permissions[]`, `resources{}` |
| `GET` | `/api/user/check-permission` | Authenticated | Query: `resource`, `action` | `has_permission` (boolean) |

---

## Role Session Policies

*Session duration and security settings per role.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/roles/session-policies` | Authenticated with `security:manage_session_policy` permission | N/A | List of Role Session Policies |
| `GET` | `/api/roles/{id}/session-policy` | Authenticated with `security:manage_session_policy` permission | N/A | `id`, `role_id`, `role_name`, `session_duration_minutes`, `refresh_token_duration_minutes`, `max_concurrent_sessions`, `idle_timeout_minutes`, `require_2fa`, `is_active` |
| `PUT` | `/api/roles/{id}/session-policy` | Authenticated with `security:manage_session_policy` permission | `session_duration_minutes` (5-1440), `refresh_token_duration_minutes` (60-525600), `max_concurrent_sessions`*, `idle_timeout_minutes`*, `require_2fa` | Session Policy Object |

---

## Role Password Policies

*Password requirements per role.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/roles/password-policies` | Authenticated with `security:manage_password_policy` permission | N/A | List of Role Password Policies (summary) |
| `GET` | `/api/roles/{id}/password-policy` | Authenticated with `security:manage_password_policy` permission | N/A | `id`, `role_id`, `role_name`, `min_length`, `max_length`, `require_uppercase`, `require_lowercase`, `require_numbers`, `require_special`, `allowed_special_chars`, `max_age_days`, `history_count`, `min_age_hours`, `min_unique_chars`, `no_username_in_password`, `no_common_passwords`, `description`, `is_active` |
| `PUT` | `/api/roles/{id}/password-policy` | Authenticated with `security:manage_password_policy` permission | `min_length`, `max_length`, `require_uppercase`, `require_lowercase`, `require_numbers`, `require_special`, `allowed_special_chars`*, `max_age_days`*, `history_count`*, `min_age_hours`*, `min_unique_chars`*, `no_username_in_password`, `no_common_passwords`, `description`* | Password Policy Object |
| `DELETE` | `/api/roles/{id}/password-policy` | Authenticated with `security:manage_password_policy` permission | N/A | `message` |

---

## Settings & System

*Global settings, branding, and labels.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/settings` | Authenticated with `settings:read` permission | N/A | Key-value map: `branding.app_name`, `branding.logo`, `labels.client`, `labels.affiliate`, `labels.category`, `labels.subcategory`, `labels.contract`, etc. |
| `PUT` | `/api/settings` | Authenticated with `settings:update` permission | `settings` (key-value map) | `message` |

**Setting Validation:**

- Max 2000 chars per value (1MB for branding images with `branding.` prefix)
- XSS patterns blocked (`<script>`, `javascript:`, etc.)
- Color values must be valid hex (`#RGB` or `#RRGGBB`)

---

## Security Configuration

*Global security and password policy settings.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/settings/security` | Authenticated with `security:manage_lock_policy` or `security:manage_password_policy` or `security:manage_session_policy` or `security:manage_rate_limit` permissions | N/A | `lock_level_1_attempts`, `lock_level_1_duration`, `lock_level_2_attempts`, `lock_level_2_duration`, `lock_level_3_attempts`, `lock_level_3_duration`, `lock_manual_attempts`, `password_min_length`, `password_require_upper`, `password_require_lower`, `password_require_numbers`, `password_require_special`, `session_duration`, `refresh_token_duration`, `rate_limit`, `rate_burst`, `audit_retention_days`, `audit_log_reads`, `notification_email`, `notification_phone` |
| `PUT` | `/api/settings/security` | Authenticated with appropriate `security:*` permissions | Same fields as GET (all optional) | `message` |
| `GET` | `/api/settings/password-policy` | Authenticated | N/A | `min_length`, `require_upper`, `require_lower`, `require_numbers`, `require_special` |

---

## User Theme & Preferences

*User-specific theme and accessibility settings.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/user/theme` | Authenticated | N/A | `theme_preset`, `theme_mode`, `layout_mode`, `primary_color`, `secondary_color`, `background_color`, `surface_color`, `text_color`, `text_secondary_color`, `border_color`, `high_contrast`, `color_blind_mode`, `dyslexic_font`, `font_general`, `font_title`, `font_table_title` |
| `PUT` | `/api/user/theme` | Authenticated (with `theme:update` permission if not self) | `theme_mode` (light/dark/system), `layout_mode` (standard/full/centralized), `theme_preset`*, `primary_color`*, `secondary_color`*, `background_color`*, `surface_color`*, `text_color`*, `text_secondary_color`*, `border_color`*, `high_contrast`*, `color_blind_mode` (none/protanopia/deuteranopia/tritanopia), `dyslexic_font`*, `font_general`*, `font_title`*, `font_table_title`* | User Theme Object |
| `GET` | `/api/settings/theme-permissions` | Authenticated with `theme:manage_permissions` permission | N/A | `users_can_edit_theme`, `admins_can_edit_theme` |
| `PUT` | `/api/settings/theme-permissions` | Authenticated with `theme:manage_permissions` permission | `users_can_edit_theme`, `admins_can_edit_theme` | `message` |
| `GET` | `/api/settings/global-theme` | Authenticated with `theme:manage_global` permission | N/A | Global theme defaults (same fields as user theme) |
| `PUT` | `/api/settings/global-theme` | Authenticated with `theme:manage_global` permission | `theme_preset`*, `primary_color`*, `secondary_color`*, `background_color`*, `surface_color`*, `text_color`*, `text_secondary_color`*, `border_color`*, `font_general`*, `font_title`*, `font_table_title`* | Global Theme Object |
| `GET` | `/api/settings/allowed-themes` | Authenticated | N/A | `allowed_themes` (array of theme preset names) |
| `PUT` | `/api/settings/allowed-themes` | Authenticated with `theme:manage_permissions` permission | `allowed_themes` (array of strings) | `message` |
| `GET` | `/api/settings/system-config` | Authenticated with `settings:read` permission | N/A | `login_block_time`, `login_attempts`, `notification_email`, `notification_phone` |
| `PUT` | `/api/settings/system-config` | Authenticated with `settings:update` permission | `login_block_time`*, `login_attempts`*, `notification_email`*, `notification_phone`* | `message` |

---

## Dashboard Configuration

*Dashboard display settings.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/system-config/dashboard` | Authenticated with `dashboard:read` permission | N/A | `show_birthdays`, `birthdays_days_ahead`, `show_recent_activity`, `recent_activity_count`, `show_statistics`, `show_expiring_contracts`, `expiring_days_ahead`, `show_quick_actions` |
| `PUT` | `/api/system-config/dashboard` | Authenticated with `dashboard:configure` permission | `show_birthdays`, `birthdays_days_ahead` (1-90), `show_recent_activity`, `recent_activity_count` (5-50), `show_statistics`, `show_expiring_contracts`, `expiring_days_ahead` (7-180), `show_quick_actions` | `success`, `message` |

---

## Dashboard Data

*Aggregated counts for the main dashboard display.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/dashboard/counts` | Authenticated | Optional Query: `expiring_days` (default 30) | `clients{total, active, inactive, archived}`, `contracts{total, active, expiring, expired, not_started, archived}`, `categories{total, active, archived, subcategories}` |

---

## Audit Logs

*Security and operation logs.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/audit-logs` | Authenticated with `audit_logs:read` permission (or `audit_logs:read_all` for full access) | Query (all optional): `resource`, `operation`, `admin_id`, `admin_search`, `resource_id`, `resource_search`, `changed_data`, `status`, `ip_address`, `start_date` (RFC3339), `end_date` (RFC3339), `limit` (max 1000), `offset` | `data[]` (audit logs), `total`, `limit`, `offset` |
| `GET` | `/api/audit-logs/{id}` | Authenticated with `audit_logs:read` permission | N/A | `id`, `timestamp`, `operation`, `resource`, `resource_id`, `admin_id`, `admin_username`, `old_value`, `new_value`, `status`, `error_message`, `ip_address`, `user_agent`, `request_method`, `request_path`, `request_id`, `response_code`, `execution_time_ms` |
| `GET` | `/api/audit-logs/resource/{resource}/{resourceID}` | Authenticated with `audit_logs:read` permission | Query: `limit`*, `offset`* | `data[]`, `resource`, `resource_id`, `limit`, `offset` |
| `GET` | `/api/audit-logs/export` | Authenticated with `audit_logs:export` permission | Query (all optional): `resource`, `operation`, `admin_id`, `admin_search`, `resource_search`, `changed_data` | JSON file download (max 10000 records) |

**Operations:** `create`, `update`, `delete`, `login`, `archive`, `unarchive`, `upload`  
**Resources:** `user`, `client`, `affiliate`, `contract`, `category`, `subcategory`, `auth`, `settings_branding`, `settings_labels`, `settings_system`, `role_session_policy`, `role_password_policy`, `dashboard_config`, `file`  
**Status:** `success`, `error`, `failed`

---

## File Upload

*File upload for images.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `POST` | `/api/upload` | Authenticated with `settings:manage_uploads` permission | Form-Data: `file` (max 15MB) | `url` |

**Allowed MIME types:** `image/jpeg`, `image/png`, `image/gif`, `image/webp`, `image/svg+xml`

**Static Files:** Uploaded files served at `/uploads/{filename}`

---

## Deploy Configuration

*Runtime configuration for deployment.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `POST` | `/api/deploy/config` | Deploy Token (Bearer) - Bypasses permission system | `server_host`*, `server_port`*, `database_host`*, `database_port`*, `database_name`*, `database_user`*, `database_password`*, `database_ssl_mode`*, `jwt_secret_key`*, `jwt_expiration_time`*, `jwt_refresh_expiration_time`*, `security_password_min_length`*, `security_password_require_uppercase`*, `security_password_require_lowercase`*, `security_password_require_numbers`*, `security_password_require_special`*, `security_max_failed_attempts`*, `security_lockout_duration_minutes`*, `app_env`* | `success`, `message`, `errors[]`, `config{}` |
| `GET` | `/api/deploy/config/defaults` | Public - No authentication required | N/A | `success`, `config{}` (server, database, jwt settings) |
| `GET` | `/api/deploy/status` | Public - No authentication required | N/A | `status`, `message`, `environment`, `version`, `config_loaded`, `timestamp` |
| `POST` | `/api/deploy/validate` | Public - No authentication required | Same as `/api/deploy/config` | `success`, `message`, `errors[]` |

**Deploy Token:** Set via `DEPLOY_TOKEN` environment variable. Send as `Authorization: Bearer <token>`.

---

## System Initialization

*Initial system setup when database is empty.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/api/initialize/status` | Public - No authentication required | N/A | `is_initialized`, `has_database`, `database_empty`, `requires_setup`, `message`, `database_status` (empty/has_data/connected/error), `tables_with_data[]` |
| `POST` | `/api/initialize/admin` | Public (Empty DB Only) - No authentication required | `display_name`, `password` (min 24 chars), `username`* (defaults to "root") | `success`, `message`, `admin_id`, `admin_username` |

**⚠️ SECURITY:** `/api/initialize/admin` is ONLY accessible when the database is COMPLETELY EMPTY. Once any data exists, this endpoint returns 403 Forbidden.

---

## Health Check

*System health monitoring.*

| Method | Endpoint | Authentication | Required Parameters | Response Fields |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/health` | Public - No authentication required | N/A | `status` (healthy/unhealthy), `message`*, `timestamp` |

---

## Common Response Formats

### Success Response

```json
{
  "message": "Operation successful",
  "data": { ... }
}
```

### Error Response

```json
{
  "error": "Error message description"
}
```

### HTTP Status Codes

| Code | Description |
| :--- | :--- |
| `200` | Success |
| `201` | Created |
| `204` | No Content (Delete success) |
| `400` | Bad Request (Validation error) |
| `401` | Unauthorized (Missing/Invalid token) |
| `403` | Forbidden (Insufficient permissions) |
| `404` | Not Found |
| `405` | Method Not Allowed |
| `500` | Internal Server Error |
| `503` | Service Unavailable |

---

## Authentication Header

For authenticated endpoints, include the JWT token:

```http
Authorization: Bearer <jwt_token>
```

---

## Rate Limiting

- Default: 100 requests per minute
- Burst: 20 requests

Response headers:

- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

---

## Security Features

- **RBAC Permission System:** Granular permissions control access to resources (see Permission System section)
- **CORS:** Enabled for configured origins
- **Security Headers:** `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Strict-Transport-Security`, `Content-Security-Policy`
- **Request Logging:** All requests are logged with user info, IP, and execution time
- **Audit Trail:** All data modifications are logged to audit_logs table
