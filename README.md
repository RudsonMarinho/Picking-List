# Picking List

Sistema de gestão de estoque e separação de pedidos (picking list) para armazém/depósito.

## Stack

- **Backend**: Node.js + Express + Prisma + PostgreSQL, autenticação via JWT
- **Frontend**: React + Vite + React Router

## Funcionalidades

- Autenticação de usuários (admin/operador)
- Cadastro de produtos e localizações de armazém
- Controle de estoque por produto/localização
- Criação de pedidos com múltiplos itens
- Geração automática de picking list a partir de um pedido, alocando estoque disponível entre localizações
- Confirmação de separação por item, com baixa automática de estoque
- Fechamento automático do pedido quando toda a picking list é separada

## Estrutura

```
backend/    API REST (Express + Prisma)
frontend/   Aplicação web (React + Vite)
```

## Como rodar localmente

### Opção 1 — Docker (recomendado)

Só precisa ter o [Docker](https://docs.docker.com/get-docker/) instalado — não é necessário instalar Node.js nem PostgreSQL na sua máquina.

```bash
docker compose up --build
```

Isso sobe três serviços:

- **postgres** — banco de dados (porta `5432`)
- **backend** — API Express, criando/atualizando as tabelas e o usuário de exemplo automaticamente a cada start (porta `3333`)
- **frontend** — aplicação React com hot-reload (porta `5173`)

Depois de subir, acesse **http://localhost:5173** e faça login com:

- E-mail: `admin@pickinglist.com`
- Senha: `admin123`

Alterações nos arquivos de `backend/` e `frontend/` são refletidas automaticamente (volumes montados + nodemon/Vite). Para parar tudo: `Ctrl+C` ou `docker compose down` (use `docker compose down -v` se quiser apagar também os dados do banco).

### Opção 2 — Node.js local

Use esta opção se preferir rodar backend e frontend diretamente na sua máquina (requer Node.js 20+ instalado). O banco ainda roda em Docker.

#### 1. Banco de dados

```bash
docker compose up -d postgres
```

#### 2. Backend

```bash
cd backend
cp .env.example .env
npm install
npm run prisma:migrate
npm run prisma:seed
npm run dev
```

A API sobe em `http://localhost:3333`. Usuário de exemplo criado pelo seed: `admin@pickinglist.com` / `admin123`.

#### 3. Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

A aplicação sobe em `http://localhost:5173` e usa proxy para `/api` -> backend.

## Fluxo de uso

1. Cadastre produtos em **Produtos**.
2. Cadastre localizações e ajuste o estoque em **Estoque**.
3. Crie um pedido em **Pedidos** com os itens desejados.
4. Gere a picking list do pedido (aloca estoque disponível entre localizações).
5. Em **Picking Lists**, abra a lista gerada e confirme a separação de cada item — o estoque é baixado automaticamente e o pedido é concluído quando tudo for separado.
