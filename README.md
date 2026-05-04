# Fidel Quick

Plataforma modular de fidelidad con bot de WhatsApp, panel de administracion y API REST.

## Arquitectura

```
fidel-quick/
  main.go                    # Entry point, DI, adapters
  api/
    router.go                # Gin router, middleware, rutas
    middleware/               # JWT + Bearer auth
  internal/
    admin/                   # Auth (login, registro, onboarding de negocio)
    onboarding/              # Estado del proceso de onboarding (dominio independiente)
    config/                  # Env vars
    flow/                    # Motor de flujos conversacionales (WhatsApp)
    landing/                 # Pagina publica /unirse/:slug
    loyalty/                 # Registry de modulos, tipos compartidos
    modules/
      earnburn/              # Modulo de puntos (earn_burn)
      cashback/              # Modulo de cashback
    resolver/                # Resolucion de negocio y rol por telefono
    session/                 # Sesiones de usuario (Redis, TTL 30min)
    platform/
      whatsapp/              # Webhook handler, client API
      ai/                    # Gemini (procesamiento de facturas)
      cache/                 # Redis
      db/                    # PostgreSQL
      storage/               # S3/MinIO
  admin/                     # Frontend React (Vite + TailwindCSS)
  migrations/                # SQL migrations (golang-migrate)
```

## Dominios

| Dominio | Tabla(s) | Descripcion |
|---------|----------|-------------|
| **customer** | `customers` | Negocio/empresa que usa la plataforma |
| **client** | `clients` | Consumidor final del negocio |
| **collaborator** | `collaborators` | Empleado que opera el programa via WhatsApp |
| **program** | `programs`, `cashback_programs` | Programa de fidelidad del negocio |
| **reward** | `rewards`, `cashback_rewards` | Recompensa canjeable dentro de un programa |
| **onboarding** | `onboarding` | Estado del proceso de configuracion inicial |
| **admin** | `admins` | Cuenta de acceso al panel de administracion |

## Modulos de fidelidad

El sistema tiene dos modulos independientes que se registran en el `loyalty.Registry`:

### earn_burn (Puntos)
- Clientes acumulan puntos por compras
- Canjean puntos por recompensas
- `points_ratio`: cuantos puntos por unidad de compra

### cashback
- Clientes acumulan saldo en pesos por compras
- Canjean saldo por recompensas
- `cashback_rate`: porcentaje de cashback sobre el monto

### Limitaciones actuales

- **Un programa activo por tipo por negocio.** Cada customer puede tener maximo 1 programa earn_burn y 1 programa cashback activo simultaneamente. `GetProgram(customerID)` retorna un solo resultado. Si existen multiples programas activos del mismo tipo, el comportamiento es indeterminado.
- **Sin seleccion de programa.** No hay UI ni flujo de WhatsApp para elegir entre multiples programas del mismo tipo.

## Diagrama ER

Ver el diagrama completo de la base de datos en [`docs/erd.mmd`](docs/erd.mmd).

Para visualizarlo: abrir en [mermaid.live](https://mermaid.live) o usar la extension Mermaid Preview en VS Code.

## Naming conventions

| Concepto | Go | TypeScript | DB | API JSON |
|----------|-----|-----------|-----|----------|
| Tipo de programa puntos | `earn_burn` | `"earn_burn"` | `type = 'earn_burn'` | `"earn_burn"` |
| Tipo de programa cashback | `cashback` | `"cashback"` | `type = 'cashback'` | `"cashback"` |
| Recompensa (ambos modulos) | `Reward` / `CashbackReward` | `Reward` / `CashbackReward` | `rewards` / `cashback_rewards` | `reward` |
| Prefix routing WhatsApp | `reward:` (earn_burn) / `cb_reward:` (cashback) | — | — | — |

## Quick Start

### Prerequisitos
- Go 1.25+
- Node.js 20+
- Docker (para Postgres, Redis, MinIO)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### Desarrollo local

```bash
# 1. Levantar infraestructura
make infra-up

# 2. Correr migraciones
make migrate-up

# 3. Backend + Frontend
make dev-all
```

O todo junto:
```bash
make start-all
```

### URLs

| Servicio | URL |
|----------|-----|
| Admin (frontend) | http://localhost:5173 |
| API (backend) | http://localhost:8080 |
| API Docs (Swagger) | http://localhost:8080/api/docs |
| MinIO Console | http://localhost:9001 |

## Variables de entorno

Ver `.env.example` para la lista completa.
