-- Tabel settings untuk menyimpan konfigurasi global
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Tabel notifications untuk log semua notifikasi terkirim
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel VARCHAR(50) NOT NULL, -- 'telegram', 'whatsapp'
    notification_type VARCHAR(100) NOT NULL, -- 'storage_alert', 'backup_failed', 'daily_summary'
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    payload TEXT NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    delivery_status VARCHAR(20) NOT NULL, -- 'sent', 'failed'
    error TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index untuk query performance
CREATE INDEX IF NOT EXISTS idx_notifications_channel ON notifications(channel);
CREATE INDEX IF NOT EXISTS idx_notifications_sent_at ON notifications(sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_server_id ON notifications(server_id);

-- ponytail: encryption skipped - token stored plaintext in settings.value JSON
-- Upgrade path: add encrypted_value BYTEA column, migrate from value, drop value
