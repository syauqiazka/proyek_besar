# Database Migrations — Archon Backend

Migrations untuk Supabase PostgreSQL database.

## Setup Awal

1. Login ke Supabase dashboard: https://supabase.com/dashboard
2. Buka project Archon (ref: `ursjstzwelwufrwpffkm`, region: `ap-northeast-2`)
3. Sidebar kiri: **SQL Editor**

## Apply Migration

### Migration 001: Init Schema (12 tabel core)

1. Copy semua content dari `001_init_schema.sql`
2. Paste ke SQL Editor
3. Klik **Run** (atau Ctrl+Enter)
4. Verifikasi: sidebar kiri **Table Editor** — harus muncul 12 tabel:
   - tenants
   - users
   - servers
   - server_connections
   - ssh_command_logs
   - agent_api_keys
   - storage_snapshots
   - disk_usage_breakdowns
   - file_candidates
   - cleanup_actions
   - backups
   - backup_verifications
   - notifications

### Migration 002: Notification Tables (sudah applied sebelumnya)

Skip jika tabel `notifications` sudah ada dari migration 001.

## Connection String

Backend butuh env var:

```bash
DATABASE_URL=postgresql://postgres.[PROJECT_REF]:[PASSWORD]@aws-0-ap-northeast-2.pooler.supabase.com:6543/postgres
```

Dapatkan dari Supabase dashboard:
- **Settings** → **Database** → **Connection string** → **URI**
- Replace `[YOUR-PASSWORD]` dengan password DB

## Troubleshooting

**Error: relation already exists**
- Tabel sudah ada. Skip migration atau drop table dulu (hati-hati data hilang).

**Error: permission denied**
- User Supabase default (`postgres`) punya akses penuh. Pastikan login dengan role yang benar.

**Error: password authentication failed**
- Reset password di **Settings** → **Database** → **Reset Database Password**

## Next Steps

Setelah schema applied:
1. Setup backend Go connection (`internal/db/connection.go`)
2. Seed data test (opsional): tenant + user dummy
3. Test API endpoints CRUD servers
