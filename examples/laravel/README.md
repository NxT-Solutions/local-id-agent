# Laravel example

PHP [Laravel 12](https://laravel.com/) API implementing the LocalID `/localid/challenge` and `/localid/verify` contract.

## Prerequisites

- PHP 8.2+ and [Composer](https://getcomposer.org/)
- LocalID Agent running (`services/agent`)

## Setup

```bash
cd examples/laravel
composer install
cp .env.example .env
php artisan key:generate
touch database/database.sqlite
```

## Run

```bash
php artisan serve --port=8000
```

## Test with React or Vue example

```bash
# Terminal 1 — agent
cd services/agent && go run ./cmd/localid-agent --config config.example.json

# Terminal 2 — this API
cd examples/laravel && php artisan serve --port=8000

# Terminal 3 — frontend (from repo root)
pnpm run dev:react   # or: pnpm run dev:vue
```

Set `VITE_BACKEND_URL=http://localhost:8000` in the frontend `.env`.
