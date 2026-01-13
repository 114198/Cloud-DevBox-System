# Cloud DevBox Database

## Overview

Cloud DevBox uses PostgreSQL as the primary database and Redis for caching.

## Directory Structure

```
database/
├── migrations/          # SQL migration files
│   ├── 001_initial_schema.sql
│   ├── 002_environments.sql
│   ├── 003_collaboration.sql
│   ├── 004_deployments.sql
│   ├── 005_billing.sql
│   └── 006_audit_logs.sql
├── seeds/               # Seed data
│   └── 001_templates.sql
├── redis/               # Redis configuration
│   └── cache-strategy.md
└── README.md
```

## Running Migrations

### Using psql

```bash
# Connect to database
psql -h localhost -U devbox -d devbox

# Run migrations in order
\i migrations/001_initial_schema.sql
\i migrations/002_environments.sql
\i migrations/003_collaboration.sql
\i migrations/004_deployments.sql
\i migrations/005_billing.sql
\i migrations/006_audit_logs.sql

# Run seeds
\i seeds/001_templates.sql
```

### Using Docker

```bash
# Start PostgreSQL
docker run -d \
  --name devbox-postgres \
  -e POSTGRES_USER=devbox \
  -e POSTGRES_PASSWORD=devbox \
  -e POSTGRES_DB=devbox \
  -p 5432:5432 \
  postgres:16-alpine

# Run migrations
docker exec -i devbox-postgres psql -U devbox -d devbox < migrations/001_initial_schema.sql
```

## Schema Overview

### Core Tables
- `users` - User accounts
- `organizations` - Organization/team management
- `templates` - Environment templates
- `environments` - Development environments
- `projects` - Project management

### Collaboration Tables
- `collaboration_sessions` - Real-time collaboration sessions
- `collaboration_participants` - Session participants
- `meetings` - Video/audio meetings
- `meeting_participants` - Meeting attendees

### Billing Tables
- `usage_records` - Resource usage tracking
- `invoices` - Monthly invoices
- `payments` - Payment records
- `subscriptions` - User subscriptions

### Security Tables
- `audit_logs` - System audit trail
- `api_keys` - API key management
- `webhooks` - Webhook configurations

## Indexes

All tables have appropriate indexes for:
- Primary keys (UUID)
- Foreign key relationships
- Frequently queried columns
- Status/phase columns (using GIN for JSONB)
- Timestamp columns for time-based queries

## Backup Strategy

- Full backup: Daily at 2:00 AM UTC
- Incremental backup: Every hour
- Retention: 30 days
- Cross-region replication: Enabled
