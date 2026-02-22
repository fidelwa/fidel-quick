# Cashback Loyalty System — Implementation Plan

**Date:** 2026-02-21
**Scope:** Modulo cashback aislado, mismo patron que earn-burn, registrado en el Module Registry
**Prerequisito:** Modulo earn-burn implementado + arquitectura por capas (apperror, resolver.Repository)

---

## Architecture Overview

El modulo cashback sigue exactamente el mismo patron arquitectonico que earn-burn:

```
WhatsApp Message
  → webhook.Receive() (compartido)
  → Resolver contexto (compartido)
  → Flow Engine (compartido) → Registry.Dispatch() → cashback.Module.HandleCommand()
  → cashback.Service → cashback.Repository → PostgreSQL (tablas cashback_*)
  → Respuesta via WhatsApp Client (compartido)
```

```
REST API Request
  → Bearer token auth middleware (compartido)
  → apperror.ErrorHandler middleware (compartido)
  → /api/v1/cashback-programs/... routes
  → cashback.APIHandler → cashback.Service → cashback.Repository
  → JSON response
```

**Principio de aislamiento:** El modulo cashback NO importa nada del modulo earn-burn. Comparten:
- Infraestructura: DB connection, Redis, S3, WhatsApp client, AI client
- Framework: loyalty.Module interface, Registry, Flow Engine, Session, Resolvers
- Manejo de errores: apperror
- Tablas base: customers, clients, collaborators (NO se duplican)

---

## Project Structure (solo archivos nuevos)

```
fidel-quick/
├── internal/
│   └── modules/
│       └── cashback/                    # Modulo Cashback (aislado)
│           ├── module.go                # Implements loyalty.Module
│           ├── service.go               # Logica de negocio cashback
│           ├── repository.go            # Repository interface + Postgres impl
│           ├── cache.go                 # Redis operations (OTP cashback)
│           ├── api.go                   # REST API handlers (admin CRUD)
│           ├── menus.go                 # Menu definitions + flow definitions
│           └── types.go                 # Tipos del modulo
├── migrations/
│   ├── 000009_create_cashback_programs.up.sql
│   ├── 000009_create_cashback_programs.down.sql
│   ├── 000010_create_cashback_core.up.sql
│   ├── 000010_create_cashback_core.down.sql
│   ├── 000011_create_cashback_rewards.up.sql
│   └── 000011_create_cashback_rewards.down.sql
└── main.go                              # Agregar: cbModule + registry.Register(cbModule)
```

**Archivos nuevos:** 7 (modulo) + 6 (migraciones) = 13
**Archivos modificados:** 1 (main.go — solo agregar wiring del nuevo modulo)

---

## Module Interface (misma que earn-burn)

```go
// internal/modules/cashback/module.go
package cashback

type Module struct {
    service *Service
    api     *APIHandler
}

func (m *Module) Name() string { return "cashback" }
func (m *Module) Menus() map[string][]loyalty.MenuDefinition { ... }
func (m *Module) HandleCommand(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) { ... }
func (m *Module) FlowDefinitions() map[string]loyalty.FlowDefinition { ... }
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) { ... }
```

---

## Database Schema

### Migration 009: Cashback Programs

```sql
CREATE TABLE cashback_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    type VARCHAR(50) NOT NULL DEFAULT 'cashback',
    name VARCHAR(255) NOT NULL,
    cashback_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0500,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, type),
    CHECK (cashback_rate > 0 AND cashback_rate <= 1)
);
```

**Design notes:**
- `cashback_rate` es DECIMAL(5,4): soporta hasta 1.0000 (100%). Ej: 0.0500 = 5%
- CHECK constraint: rate debe ser > 0 y <= 1
- UNIQUE por (customer_id, type): un negocio solo puede tener 1 programa cashback

### Migration 010: Cashback Core (Balances + Transactions)

```sql
CREATE TABLE cashback_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    balance DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(client_id, program_id)
);

CREATE TABLE cashback_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    collaborator_id UUID REFERENCES collaborators(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('earn', 'burn', 'adjustment')),
    amount DECIMAL(12,2) NOT NULL,
    purchase_amount DECIMAL(12,2),
    balance_after DECIMAL(12,2) NOT NULL,
    invoice_url TEXT,
    description TEXT,
    manual_entry BOOLEAN NOT NULL DEFAULT false,
    correction_reason TEXT,
    correction_evidence_url TEXT,
    correctable_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cb_transactions_client ON cashback_transactions(client_id);
CREATE INDEX idx_cb_transactions_created ON cashback_transactions(created_at);
```

**Design notes:**
- `balance` es DECIMAL(12,2) — pesos con centavos. CHECK >= 0
- `purchase_amount` almacena el monto original de la factura (para recalcular en correcciones)
- `manual_entry` flag para auditoria (igual que earn-burn)
- `amount` puede ser negativo (burn, adjustment negativo)

### Migration 011: Cashback Rewards & Redemptions

```sql
CREATE TABLE cashback_rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cost DECIMAL(12,2) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cb_rewards_customer ON cashback_rewards(customer_id);

CREATE TABLE cashback_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    reward_id UUID NOT NULL REFERENCES cashback_rewards(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    code VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'confirmed', 'expired', 'cancelled')),
    amount_spent DECIMAL(12,2) NOT NULL,
    confirmed_by UUID REFERENCES collaborators(id),
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cb_redemptions_code ON cashback_redemptions(code);
CREATE INDEX idx_cb_redemptions_client ON cashback_redemptions(client_id);
CREATE INDEX idx_cb_redemptions_status ON cashback_redemptions(status) WHERE status = 'pending';
```

---

## Menus y Flows

### Client Menu (5 opciones)

| Menu Option | Command ID | Tipo |
|-------------|-----------|------|
| Consultar saldo | `cb_check_balance` | Ejecucion directa |
| Ver beneficios | `cb_list_rewards` | Ejecucion directa (texto catalogo) |
| Canjear beneficio | `cb_redeem` | Ejecucion directa (lista interactiva filtrada) |
| Cargar cashback | `cb_load_request` | Ejecucion directa (genera codigo) |
| Dejar feedback | `submit_feedback` | Flujo: pedir comentario (compartido con earn-burn) |

### Collaborator Menu (5 opciones)

| Menu Option | Command ID | Flow Steps |
|-------------|-----------|------------|
| Acreditar cashback | `cb_add_cashback` | 1. Pedir OTP → 2. Pedir foto ticket |
| Consultar saldo de cliente | `cb_list_balance` | 1. Pedir OTP → 2. Mostrar saldo |
| Confirmar canje | `cb_confirm_redemption` | 1. Pedir codigo → 2. Confirmar |
| Corregir transaccion | `cb_update_cashback` | 1. Pedir OTP → 2. Seleccionar tx → 3. Nuevo monto factura → 4. Evidencia → 5. Comentario |
| Procesar carga | `cb_load_process` | 1. Pedir codigo → 2. Pedir foto ticket |

### Prefijo de seleccion

El flow engine detecta `benefit:` como prefijo (igual que `reward:` en earn-burn) para iniciar el flujo de canje con el reward_id pre-cargado.

---

## Service Layer

```go
// internal/modules/cashback/service.go
type Service struct {
    repo  Repository
    cache Cache
    log   *slog.Logger
}

// Operaciones WhatsApp
func (s *Service) GetProgram(ctx context.Context, customerID string) (*CashbackProgram, error)
func (s *Service) CheckBalance(ctx context.Context, clientID, programID string) (float64, error)
func (s *Service) ListTransactions(ctx context.Context, clientID, programID string, limit int) ([]CashbackTransaction, error)
func (s *Service) AddCashback(ctx context.Context, req AddCashbackReq) (*CashbackTransaction, error)
func (s *Service) UpdateCashback(ctx context.Context, req UpdateCashbackReq) (*CashbackTransaction, error)
func (s *Service) ListRewards(ctx context.Context, customerID, programID string, maxCost float64) ([]CashbackReward, error)
func (s *Service) RequestRedemption(ctx context.Context, req CashbackRedemptionReq) (*CashbackRedemption, string, error)
func (s *Service) ConfirmRedemption(ctx context.Context, code, collaboratorID string) (*CashbackRedemption, error)
func (s *Service) RequestLoadCode(ctx context.Context, clientID, customerID string) (string, error)
func (s *Service) ValidateLoadCode(ctx context.Context, code string) (*OTPData, error)
func (s *Service) GetClientName(ctx context.Context, clientID string) (string, error)
func (s *Service) GetReward(ctx context.Context, rewardID string) (*CashbackReward, error)

// Operaciones Admin CRUD
func (s *Service) ListPrograms(ctx context.Context, customerID string) ([]CashbackProgram, error)
func (s *Service) CreateProgram(ctx context.Context, req CreateCashbackProgramReq) (*CashbackProgram, error)
func (s *Service) UpdateProgram(ctx context.Context, id string, req UpdateCashbackProgramReq) (*CashbackProgram, error)
func (s *Service) ListAllRewards(ctx context.Context, customerID, programID string) ([]CashbackReward, error)
func (s *Service) CreateRewardAdmin(ctx context.Context, req CreateCashbackRewardReq) (*CashbackReward, error)
func (s *Service) UpdateRewardAdmin(ctx context.Context, id string, req UpdateCashbackRewardReq) (*CashbackReward, error)
```

**Diferencia clave en calculo:**
```go
// Earn-burn: puntos = floor(monto / ratio)
points := int(math.Floor(amount / float64(program.PointsRatio)))

// Cashback: cashback = floor(monto * rate * 100) / 100
cashback := math.Floor(amount * program.CashbackRate * 100) / 100
```

---

## API Handlers (REST)

```
POST   /api/v1/cashback-programs                                → CreateProgram
GET    /api/v1/cashback-programs                                → ListPrograms
PUT    /api/v1/cashback-programs/:id                            → UpdateProgram
POST   /api/v1/cashback-programs/:program_id/rewards            → CreateReward
GET    /api/v1/cashback-programs/:program_id/rewards             → ListRewards
PUT    /api/v1/cashback-programs/:program_id/rewards/:id        → UpdateReward
DELETE /api/v1/cashback-programs/:program_id/rewards/:id        → DeleteReward (soft)
GET    /api/v1/cashback-programs/:program_id/clients/:id/balance       → GetClientBalance
GET    /api/v1/cashback-programs/:program_id/clients/:id/transactions  → ListClientTransactions
```

Todos los handlers siguen el patron: parsear request → llamar service → `c.Error(err)` o `c.JSON(200, result)`. No hacen SQL directo.

---

## OTP Types (Redis)

El modulo cashback usa el mismo sistema OTP unificado pero con tipos diferentes para evitar colisiones:

| Codigo | type | Proposito | TTL | Uso |
|--------|------|-----------|-----|-----|
| OTP identidad | `identity` | Colaborador identifica al cliente | 15 min | Multi-uso (GET) — **compartido** |
| Codigo canje cashback | `cb_redemption` | Reclamar beneficio | 1h | Un solo uso (GETDEL) |
| Codigo carga cashback | `cb_load_points` | Vincular carga a cliente | 15min | Un solo uso (GETDEL) |

**Nota:** El OTP de identidad (`type: "identity"`) es compartido entre earn-burn y cashback. El mismo codigo sirve para ambos modulos ya que identifica al cliente, no al programa.

---

## Types

```go
// internal/modules/cashback/types.go
type CashbackProgram struct {
    ID           string
    CustomerID   string
    Type         string
    Name         string
    CashbackRate float64  // 0.05 = 5%
    Active       bool
}

type CashbackBalance struct {
    ID        string
    ClientID  string
    ProgramID string
    Balance   float64  // pesos MXN
}

type CashbackTransaction struct {
    ID                    string
    ClientID              string
    ProgramID             string
    CollaboratorID        string
    Type                  string   // "earn", "burn", "adjustment"
    Amount                float64  // cashback amount (pesos)
    PurchaseAmount        float64  // original purchase amount
    BalanceAfter          float64
    InvoiceURL            string
    ManualEntry           bool
    CorrectionReason      string
    CorrectionEvidenceURL string
    CorrectableUntil      *time.Time
    CreatedAt             time.Time
}

type CashbackReward struct {
    ID          string
    CustomerID  string
    ProgramID   string
    Name        string
    Description string
    Cost        float64  // pesos MXN
    Active      bool
}

type CashbackRedemption struct {
    ID          string
    ClientID    string
    RewardID    string
    ProgramID   string
    Code        string
    Status      string
    AmountSpent float64
    ConfirmedBy string
    ExpiresAt   time.Time
    ConfirmedAt *time.Time
    CreatedAt   time.Time
}

type AddCashbackReq struct {
    ClientID       string
    ProgramID      string
    CollaboratorID string
    PurchaseAmount float64  // monto de la factura
    InvoiceURL     string
}

type UpdateCashbackReq struct {
    TransactionID         string
    CollaboratorID        string
    NewPurchaseAmount     float64  // nuevo monto de factura (sistema recalcula cashback)
    CorrectionReason      string
    CorrectionEvidenceURL string
}

type CashbackRedemptionReq struct {
    ClientID  string
    ProgramID string
    RewardID  string
}
```

---

## Wiring en main.go

```go
// Solo se necesita agregar al final del wiring existente:

// Cashback module
cbRepo := cashback.NewPostgresRepository(database)
cbCache := cashback.NewRedisCache(redisClient)
cbService := cashback.NewService(cbRepo, cbCache, log)
cbAPI := cashback.NewAPIHandler(cbService)
cbModule := cashback.NewModule(cbService, cbAPI)
registry.Register(cbModule)
```

**Nada mas cambia en main.go.** El Registry ya despacha comandos al modulo correcto por `command_id`, el Flow Engine ya soporta `benefit:` prefix, y el router ya registra las rutas de cada modulo.

---

## Order of Implementation

### Paso 1: Migraciones
1. `000009_create_cashback_programs.up.sql`
2. `000010_create_cashback_core.up.sql`
3. `000011_create_cashback_rewards.up.sql`

### Paso 2: Types + Repository
4. `types.go` — structs y request types
5. `repository.go` — Repository interface + Postgres impl

### Paso 3: Cache + Service
6. `cache.go` — Redis operations para OTP cashback
7. `service.go` — logica de negocio con apperror

### Paso 4: Menus + Module + API
8. `menus.go` — menu definitions + flow definitions + validaciones
9. `module.go` — implements loyalty.Module, HandleCommand switch
10. `api.go` — REST handlers via service

### Paso 5: Wiring
11. `main.go` — agregar 6 lineas de wiring

### Paso 6: Verificacion
12. `go build ./...` + `go vet ./...`
13. `make dev` + probar via WhatsApp
14. Probar API REST con curl

---

## Verificacion

1. `go build ./...` compila sin errores
2. `make dev` arranca sin errores
3. Un negocio con programa cashback muestra menus de cashback (5 cliente / 5 colaborador)
4. Un negocio con AMBOS programas (earn-burn + cashback) muestra menus de ambos modulos via `FilteredMenus` + opcion "Usar otro establecimiento" al final. Un negocio con solo cashback muestra solo menus cashback
5. "Consultar saldo" muestra balance en pesos: `$350 MXN`
6. "Acreditar cashback" calcula correctamente: $2,000 * 5% = $100 MXN
7. "Canjear beneficio" filtra por saldo y presenta lista interactiva
8. "Corregir transaccion" recalcula cashback basado en nuevo monto de factura
9. API REST CRUD funciona para cashback-programs y rewards
10. Tablas cashback_* son completamente independientes de tablas earn-burn

---

## Risk Assessment

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|------------|--------|------------|
| 1 | Menu de WhatsApp excede 10 items (earn-burn + cashback combinados) | L | M | **Mitigado:** `FilteredMenus` filtra por `ActiveModules` del negocio. Solo muestra menus de modulos con programas activos + "Usar otro establecimiento". Si `ActiveModules` esta vacio (sesion legacy), hace fallback a `AllMenus` |
| 2 | Precision decimal: errores de floating point en calculos de cashback | L | M | Usar `math.Floor(amount * rate * 100) / 100` para redondeo consistente. DECIMAL(12,2) en DB |
| 3 | Confusion de usuario: menus earn-burn y cashback mezclados | L | L | **Mitigado:** `FilteredMenus` solo muestra menus de modulos activos del negocio. Prefijos claros en command IDs (`cb_`). Backfill automatico de `ActiveModules` en sesiones legacy |
| 4 | Colision de codigos OTP entre modulos | L | L | Types diferentes (`cb_redemption` vs `redemption`). Generacion con crypto/rand tiene baja probabilidad de colision |
