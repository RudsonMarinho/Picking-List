-- Estende asset e collaborator para cobrir os campos que a UI original
-- (web/index.html) já exibia — hostname/marca/modelo/specs do ativo e
-- cargo do colaborador. Todos NULLable: não quebra dados existentes nem
-- a suíte negativa (RLS/FK das tabelas não muda).
ALTER TABLE asset
    ADD COLUMN hostname VARCHAR(255) NULL,
    ADD COLUMN brand     VARCHAR(100) NULL,
    ADD COLUMN model     VARCHAR(100) NULL,
    ADD COLUMN cpu       VARCHAR(100) NULL,
    ADD COLUMN ram       VARCHAR(50)  NULL,
    ADD COLUMN storage   VARCHAR(100) NULL,
    ADD COLUMN notes     TEXT         NULL;

ALTER TABLE collaborator
    ADD COLUMN cargo VARCHAR(150) NULL;
