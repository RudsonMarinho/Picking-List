-- Código de patrimônio digitado manualmente pelo usuário na UI original
-- (campo "ID Patrimônio", ex: RDS-023) — distinto do id (UUID) real e do
-- serial_number (S/N do fabricante). Único por tenant quando preenchido.
ALTER TABLE asset ADD COLUMN asset_tag VARCHAR(50) NULL;

CREATE UNIQUE INDEX ux_asset_tag ON asset(tenant_id, asset_tag)
    WHERE asset_tag IS NOT NULL;
