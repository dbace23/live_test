CREATE TABLE IF NOT EXISTS shipments (
  id               SERIAL PRIMARY KEY,
  nama             TEXT        NOT NULL,
  pengirim         TEXT        NOT NULL,
  nama_penerima    TEXT        NOT NULL,
  alamat_penerima  TEXT        NOT NULL,
  nama_item        TEXT        NOT NULL,
  berat_item       INTEGER     NOT NULL CHECK (berat_item >= 0),
  "timestamp"      TIMESTAMPTZ NOT NULL DEFAULT NOW(),  
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_shipments_created_at ON shipments (created_at DESC)