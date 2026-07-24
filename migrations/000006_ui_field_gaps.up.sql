-- Mais campos que a UI original exibia e o schema não tinha — mesma
-- decisão do usuário da migration 000004 (estender em vez de simplificar
-- a UI). Todos NULLable/com default seguro: não quebra dados existentes.
ALTER TABLE company
    ADD COLUMN city  VARCHAR(100) NULL,
    ADD COLUMN state VARCHAR(2)  NULL;

ALTER TABLE branch
    ADD COLUMN city  VARCHAR(100) NULL,
    ADD COLUMN state VARCHAR(2)  NULL;

ALTER TABLE sector
    ADD COLUMN description TEXT NULL;

ALTER TABLE asset_type
    ADD COLUMN icon        VARCHAR(50) NULL,
    ADD COLUMN description TEXT        NULL;

ALTER TABLE collaborator
    ADD COLUMN email  CITEXT NULL,
    ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'active';
-- A UI original não coleta CPF (document) — vira opcional. Continua
-- existindo pra quem quiser preencher via import/API diretamente.
ALTER TABLE collaborator ALTER COLUMN document DROP NOT NULL;
