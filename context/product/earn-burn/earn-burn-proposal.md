# Earn-Burn Loyalty System — Implementation Plan

**Mode:** DEEP
**Date:** 2026-02-13
**Scope:** Modular monolith earn-burn system with WhatsApp interactive menus + REST API

---

## Architecture Overview

```
WhatsApp Message
  → webhook.Receive()
  → Identify user (phone → DB lookup) + business context (from landing page deeplink)
  → Load flow state (or present main menu)
  → User selects menu option → Router.Dispatch(command) → routes to correct module
  → Module.HandleCommand() → starts flow (step-by-step data collection)
  → Each step: prompt for data → validate → next step → execute business logic
  → Template response → WhatsApp reply sent

AI (solo fotos): Cuando un flujo requiere foto de ticket,
  → AI Client procesa imagen → extrae monto → continua flujo
```

```
REST API Request
  → Bearer token auth middleware
  → apperror.ErrorHandler middleware (converts AppError → JSON with correct HTTP status)
  → /api/v1/{module}/... routes
  → Module REST handlers (no SQL directo) → Service (clasifica errores con apperror) → Repository
  → JSON response (errores 4xx con mensaje amigable, 5xx sin detalles internos)
```

**Arquitectura por capas (implementada):**
```
Controller (api.go)     → Solo parseo de request + c.Error(err) + c.JSON()
Service (service.go)    → Logica de negocio + clasificacion de errores con apperror
Repository (repository.go) → SQL queries + acceso a datos
apperror middleware      → Convierte c.Error(AppError) a JSON con HTTP status correcto
resolver.Repository     → Queries SQL para resolvers, landing page y auto-registro
```

---

## Project Structure

```
fidel-quick/
├── main.go                              # Wire everything, start server
├── internal/
│   ├── config/
│   │   └── config.go                    # Centralized env config struct
│   ├── platform/                        # Shared infrastructure
│   │   ├── db/
│   │   │   └── postgres.go              # *sql.DB connection pool
│   │   ├── cache/
│   │   │   └── redis.go                 # Redis client wrapper
│   │   ├── storage/
│   │   │   └── s3.go                    # MinIO/S3 upload client
│   │   ├── whatsapp/
│   │   │   ├── client.go                # Send messages (text, interactive)
│   │   │   ├── webhook.go               # Verify + Receive handlers
│   │   │   └── types.go                 # WhatsApp API payload types
│   │   ├── ai/
│   │   │   ├── client.go                # Claude API client (SOLO procesamiento de fotos de tickets)
│   │   │   └── types.go                 # Photo processing types
│   │   └── logger/
│   │       └── logger.go                # slog structured logger setup
│   ├── landing/                         # Landing page for QR onboarding
│   │   ├── handler.go                   # GET /unirse/:slug → render landing page (uses resolver.Repository)
│   │   └── templates/
│   │       ├── join.html                # Mobile-first landing page
│   │       └── 404.html                 # Business not found page
│   ├── deeplink/
│   │   └── generator.go                 # Generate wa.me/{phone}?text=... URLs
│   ├── flow/
│   │   ├── engine.go                    # Flow engine: manages step-by-step interactive flows
│   │   ├── state.go                     # Flow state: Redis persistence of current flow + step + data
│   │   └── types.go                     # FlowDefinition, StepDefinition, FlowState types
│   ├── session/
│   │   └── manager.go                   # Redis session management
│   │                                    #   session:{phone} → {customer_id, role, user_id, business_name} TTL 30min
│   │                                    #   session:select:{phone} → pending selection options TTL 5min
│   │                                    #   flow:{phone}:{customer_id} → {current_flow, current_step, collected_data} TTL 30min
│   ├── apperror/                        # Centralized error handling
│   │   ├── apperror.go                  # AppError type + constructors (NotFound, BadRequest, Internal, Conflict)
│   │   └── middleware.go                # Gin middleware: c.Error(AppError) → JSON with correct HTTP status
│   ├── resolver/
│   │   ├── repository.go               # Repository interface + Postgres impl for resolvers/landing/auto-registro
│   │   ├── business.go                  # Resolve business context from message (uses Repository, not *sql.DB):
│   │   │                                #   1. Check active session in Redis
│   │   │                                #   2. Extract customer_id from landing page deeplink
│   │   │                                #   3. Lookup phone in collaborators/clients
│   │   │                                #   4. If multiple: present selection menu
│   │   │                                #   5. If none: ask to scan QR
│   │   └── role.go                      # Resolve role within business (uses Repository, not *sql.DB):
│   │                                    #   phone in collaborators → collaborator
│   │                                    #   phone in clients → client
│   │                                    #   collaborator takes priority if in both
│   ├── loyalty/                         # Module framework
│   │   ├── module.go                    # Module interface definition
│   │   ├── registry.go                  # Module registry + command dispatcher
│   │   └── types.go                     # Shared types (Command, CommandResult, UserContext, MenuDefinition, FlowDefinition)
│   └── modules/
│       └── earnburn/                    # Earn-Burn module
│           ├── module.go                # Implements loyalty.Module
│           ├── service.go               # Business logic (points, redemptions)
│           ├── repository.go            # Repository interface + Postgres impl
│           ├── cache.go                 # Redis operations (TTLs, redemption codes)
│           ├── api.go                   # REST API handlers (admin CRUD)
│           ├── menus.go                 # Menu definitions per role + flow definitions
│           └── types.go                 # Module-specific types
├── api/
│   ├── router.go                        # REST API route setup
│   └── middleware/
│       └── auth.go                      # Bearer token middleware
├── migrations/
│   ├── 000001_create_platform_config.up.sql
│   ├── 000001_create_platform_config.down.sql
│   ├── 000002_create_customers.up.sql
│   ├── 000002_create_customers.down.sql
│   ├── 000003_create_collaborators.up.sql
│   ├── 000003_create_collaborators.down.sql
│   ├── 000004_create_clients.up.sql
│   ├── 000004_create_clients.down.sql
│   ├── 000005_create_programs.up.sql
│   ├── 000005_create_programs.down.sql
│   ├── 000006_create_earnburn.up.sql
│   ├── 000006_create_earnburn.down.sql
│   ├── 000007_create_rewards.up.sql
│   ├── 000007_create_rewards.down.sql
│   ├── 000008_create_feedback.up.sql
│   └── 000008_create_feedback.down.sql
├── .env.example                         # Updated with new vars
├── docker-compose.yml                   # Existing (no changes)
├── Dockerfile                           # Existing (no changes)
├── Makefile                             # Existing (no changes)
├── go.mod                               # Add new dependencies
└── context/                             # Existing docs (no changes)
```

**Total new files:** ~35
**Modified files:** main.go (rewrite), go.mod (add deps), .env.example (add vars)

---

## Module Interface (Core Contract)

This is the key abstraction that enables future modules (cashback, tiers, etc.):

```go
// internal/loyalty/module.go
package loyalty

import "github.com/gin-gonic/gin"

// Module is the contract every loyalty system must implement.
// To add a new loyalty type (cashback, tiers, etc.):
// 1. Create internal/modules/{name}/
// 2. Implement this interface
// 3. Register in main.go
type Module interface {
    // Name returns the module identifier (e.g., "earn_burn")
    Name() string

    // Menus returns interactive WhatsApp menu definitions per role.
    // The registry aggregates all module menus to build the main menu.
    Menus() map[string][]MenuDefinition // role → menu options

    // HandleCommand processes a menu selection or flow step.
    HandleCommand(ctx context.Context, cmd Command) (*CommandResult, error)

    // FlowDefinitions returns step-by-step flow definitions for each command.
    FlowDefinitions() map[string]FlowDefinition // command_id → flow

    // RegisterRoutes adds this module's REST API routes.
    RegisterRoutes(rg *gin.RouterGroup)
}
```

```go
// internal/loyalty/types.go — key types
type MenuDefinition struct {
    ID          string // e.g., "check_points"
    Title       string // e.g., "Consultar puntos"
    Description string // Shown in WhatsApp list
    Role        string // "client" or "collaborator"
}

type FlowDefinition struct {
    CommandID string
    Steps     []StepDefinition
}

type StepDefinition struct {
    ID          string           // e.g., "ask_otp"
    Prompt      string           // Message sent to user
    Validate    func(string) error
    NeedsPhoto  bool             // If true, expects image message (AI processes it)
}

type Command struct {
    ID          string            // Menu option selected
    UserContext UserContext
    Data        map[string]string // Collected data from flow steps
}

type CommandOption struct {
    ID          string // e.g., "reward:{uuid}"
    Title       string // e.g., "Cafe gratis"
    Description string // e.g., "10 pts"
}

type CommandResult struct {
    Message    string           // Text message to send back to user
    Options    []CommandOption  // If set, shown as interactive list after Message
    ListHeader string           // Header for the interactive list
    Data       map[string]interface{}
}
```

```go
// internal/loyalty/registry.go
package loyalty

// Registry manages all loyalty modules.
type Registry struct {
    modules  map[string]Module
    commands map[string]string // command_id → module_name mapping
}

func (r *Registry) Register(m Module) { ... }
func (r *Registry) AllMenus(role string) []MenuDefinition { ... }
func (r *Registry) Dispatch(ctx context.Context, cmd Command) (*CommandResult, error) { ... }
func (r *Registry) RegisterAllRoutes(rg *gin.RouterGroup) { ... }
```

---

## Database Schema (MVP Adapted)

### Migration 001: Platform Config

```sql
CREATE TABLE platform_config (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Initial platform config
INSERT INTO platform_config (key, value) VALUES
    ('whatsapp_phone_number_id', ''),
    ('platform_name', 'Fidel'),
    ('platform_url', '');
```

**Design notes:**
- Key-value table for platform-level configuration
- `whatsapp_phone_number_id`: the single WhatsApp number shared by all businesses

### Migration 002: Customers (B2B Businesses)

```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    address TEXT,
    phone VARCHAR(20) NOT NULL,
    logo_url TEXT,
    description TEXT,
    welcome_message TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_customers_slug ON customers(slug);
```

**Design notes:**
- `slug`: URL-friendly identifier for landing page (`/unirse/cafe-roma`). UNIQUE
- `logo_url`: displayed on the landing page
- `description`: short description of the loyalty program shown on landing page
- No WhatsApp phone IDs — the platform uses 1 shared number (stored in `platform_config`)
- `welcome_message`: personalized welcome message per business (shown on first interaction)
- No change_log table in MVP — add later via trigger/audit table

### Migration 003: Collaborators (Business Employees)

```sql
CREATE TABLE collaborators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    hash_id VARCHAR(100) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, phone)
);

CREATE INDEX idx_collaborators_phone ON collaborators(phone);
CREATE INDEX idx_collaborators_customer ON collaborators(customer_id);
```

### Migration 004: Clients (End Users)

```sql
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    name VARCHAR(255),
    phone VARCHAR(20) NOT NULL,
    hash VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, phone)
);

CREATE INDEX idx_clients_phone ON clients(phone);
CREATE INDEX idx_clients_customer ON clients(customer_id);
```

### Migration 005: Programs (formerly loyalty_configs)

```sql
-- Loyalty program configuration per business
CREATE TABLE programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    points_ratio INTEGER NOT NULL DEFAULT 1000,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, type)
);

### Migration 006: Earn-Burn Core

-- Current points balance per client per program
CREATE TABLE points_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES programs(id),
    balance INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(client_id, program_id)
);

-- Full transaction history (earn, burn, adjustment)
CREATE TABLE points_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES programs(id),
    collaborator_id UUID REFERENCES collaborators(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('earn', 'burn', 'adjustment')),
    amount INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    invoice_url TEXT,
    description TEXT,
    correctable_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_client ON points_transactions(client_id);
CREATE INDEX idx_transactions_created ON points_transactions(created_at);
```

**Design notes:**
- `programs`: renamed from `loyalty_configs`. Added `name` column for display
- `points_balances`: denormalized current balance (avoid SUM on every query). FK to `program_id`
- `points_transactions`: immutable audit log. FK to `program_id`
- `correctable_until`: Postgres timestamp for 2h correction window (auditable)
- `balance_after`: snapshot for audit trail

### Migration 007: Rewards & Redemptions

```sql
-- Reward catalog per program
CREATE TABLE rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    program_id UUID NOT NULL REFERENCES programs(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    points_cost INTEGER NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rewards_customer ON rewards(customer_id);

-- Redemption requests with codes
CREATE TABLE redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    reward_id UUID NOT NULL REFERENCES rewards(id),
    program_id UUID NOT NULL REFERENCES programs(id),
    code VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'confirmed', 'expired', 'cancelled')),
    points_spent INTEGER NOT NULL,
    confirmed_by UUID REFERENCES collaborators(id),
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_redemptions_code ON redemptions(code);
CREATE INDEX idx_redemptions_client ON redemptions(client_id);
CREATE INDEX idx_redemptions_status ON redemptions(status) WHERE status = 'pending';
```

### Migration 008: Feedback

```sql
CREATE TABLE feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_customer ON feedback(customer_id);
```

---

## Core Components Detail

### 1. Config (`internal/config/config.go`)

Single struct loaded from env vars:

```go
type Config struct {
    Port                  string
    Env                   string
    DatabaseURL           string
    RedisURL              string
    S3Endpoint            string
    S3Bucket              string
    S3Region              string
    AWSAccessKeyID        string
    AWSSecretAccessKey    string
    AnthropicAPIKey       string // Solo para procesamiento de fotos de tickets (OCR)
    WhatsAppVerifyToken   string
    WhatsAppAPIToken      string
    WhatsAppPhoneNumberID string
    WhatsAppDisplayPhone  string // Display phone for wa.me deeplinks (e.g. "5215551234567")
    PlatformURL           string // Base URL for landing pages (e.g. "https://fidel.app")
    BearerToken           string // API auth
}
```

### 2. WhatsApp Client (`internal/platform/whatsapp/client.go`)

```go
type Client struct {
    apiToken    string
    phoneID     string
    httpClient  *http.Client
}

func (c *Client) SendText(ctx context.Context, to, text string) error
func (c *Client) SendInteractiveList(ctx context.Context, to string, list InteractiveList) error
```

### 3. WhatsApp Webhook (`internal/platform/whatsapp/webhook.go`)

Enhanced from existing code. Dependencies: `*Client`, `*session.Manager`, `*resolver.BusinessResolver`, `*resolver.RoleResolver`, `resolver.Repository`, `*flow.Engine`.

```go
type WebhookHandler struct {
    verifyToken string
    client      *Client
    session     *session.Manager
    business    *resolver.BusinessResolver
    role        *resolver.RoleResolver
    repo        resolver.Repository  // For auto-registration of new clients
    engine      *flow.Engine
    log         *slog.Logger
}

func (h *WebhookHandler) Receive(c *gin.Context) {
    // Parse payload, respond 200 immediately
    // For each message (async goroutine):
    //   1. Check session in Redis
    //   2. If no session → resolve business + role (deeplink → phone lookup → selection menu)
    //   3. On new session: ResetFlow to clear stale flow state
    //   4. Auto-register client if new (via h.repo.RegisterClient)
    //   5. Dispatch to flow engine: HandleMessage(ctx, user, msgType, msgText, imageURL)
}
```

### 4. AI Client — Solo Procesamiento de Fotos (`internal/platform/ai/client.go`)

```go
type Client struct {
    apiKey     string
    httpClient *http.Client
}

// ExtractAmountFromPhoto sends a ticket photo to Claude and extracts the total amount.
// This is the ONLY use of AI in the system — no conversational AI.
func (c *Client) ExtractAmountFromPhoto(ctx context.Context, imageURL string) (*PhotoResult, error)

type PhotoResult struct {
    Amount    float64 // Extracted amount from ticket
    Currency  string  // Detected currency
    Confident bool    // Whether the extraction is reliable
    RawText   string  // OCR text for debugging
}
```

The AI client does NOT handle conversation or tool_use loops.
It only processes ticket photos to extract amounts for the points calculation flow.
If the photo is unreadable after 3 attempts, the flow falls back to manual amount entry.

### 4b. Flow Engine (`internal/flow/engine.go`)

```go
type Engine struct {
    registry *loyalty.Registry
    session  *session.Manager
    wa       *whatsapp.Client
    ai       *ai.Client // Only used for photo processing steps
}

// HandleMessage processes an incoming message within the user's context.
// Checks for active flows, menu selections, or presents the main menu.
func (e *Engine) HandleMessage(ctx context.Context, user UserContext, msgType, msgText, imageURL string) error

// ResetFlow clears any active flow state for a user in a business.
// Called when creating a new session to prevent stale flow state.
func (e *Engine) ResetFlow(ctx context.Context, phone, customerID string)

// sendResult sends a command result (text, interactive list, or both) then shows main menu.
// If result has Options, sends as interactive list; otherwise sends text + menu.
func (e *Engine) sendResult(ctx context.Context, user UserContext, result *CommandResult) error

// startFlowWithData begins a flow with pre-collected data, skipping satisfied steps.
// Used when interactive list selection provides data (e.g., reward_id from "reward:{id}").
func (e *Engine) startFlowWithData(ctx context.Context, user UserContext, commandID string, data map[string]string) error
```

**Flow state** stored in Redis:
```
flow:{phone}:{customer_id} → {
    current_flow: "add_points",
    current_step: 1,         // index into FlowDefinition.Steps
    collected_data: {         // data gathered so far
        "otp": "ABC123",
        "amount": 1500
    },
    started_at: "2026-02-13T..."
}
TTL: 30min (reset on each interaction)
```

**Flow lifecycle:**
1. User selects menu option → `StartFlow(commandID)`
2. Engine sends first step's prompt to user
3. User responds → `ProcessStep(input)` → validate → advance step
4. If step needs photo → AI Client extracts amount → continue
5. All steps collected → execute business logic via `Module.HandleCommand()`
6. Send result message → clear flow state

### 5. Logger (`internal/platform/logger/logger.go`)

Uses Go's built-in `log/slog` (available since Go 1.21):

```go
func Setup(env string) *slog.Logger
// Development: text handler with colors
// Production: JSON handler
```

### 6. Earn-Burn Module

#### `internal/modules/earnburn/module.go`
```go
type Module struct {
    service *Service
    api     *APIHandler
}

func (m *Module) Name() string { return "earn_burn" }
func (m *Module) Menus() map[string][]loyalty.MenuDefinition { return menus }
func (m *Module) HandleCommand(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error)
func (m *Module) FlowDefinitions() map[string]loyalty.FlowDefinition { return flows }
func (m *Module) RegisterRoutes(rg *gin.RouterGroup)

// APIHandler depends on *Service (not *PostgresRepository) — layered architecture
type APIHandler struct { service *Service }
func NewAPIHandler(service *Service) *APIHandler
```

#### `internal/modules/earnburn/menus.go`

Interactive menu definitions and step-by-step flows for earn-burn operations:

**Client Menu:**

| Menu Option | Command ID | Flow Steps |
|-------------|-----------|------------|
| Consultar puntos | `check_points` | (sin pasos — ejecucion directa) |
| Ver recompensas | `list_all_rewards` | (sin pasos — catalogo completo como texto, muestra status por recompensa) |
| Canjear recompensa | `redeem_rewards` | (sin pasos — retorna lista interactiva filtrada por balance; al seleccionar inicia `request_redemption` via `startFlowWithData`) |
| Cargar puntos | `load_points_request` | (sin pasos — genera codigo) |
| Dejar feedback | `submit_feedback` | 1. Pedir comentario → 2. Guardar |

**Collaborator Menu:**

| Menu Option | Command ID | Flow Steps |
|-------------|-----------|------------|
| Agregar puntos | `add_points` | 1. Pedir OTP → 2. Validar → 3. Pedir foto ticket → 4. (AI extrae monto o manual) → 5. Confirmar |
| Consultar puntos de cliente | `list_points` | 1. Pedir OTP → 2. Validar → 3. Mostrar balance |
| Confirmar canje | `confirm_redemption` | 1. Pedir codigo → 2. Validar → 3. Confirmar |
| Corregir transaccion | `update_points` | 1. Pedir OTP → 2. Listar corregibles → 3. Seleccionar → 4. Nuevo monto → 5. Pedir evidencia → 6. Pedir comentario |
| Procesar carga de puntos | `load_points_process` | 1. Pedir codigo cliente → 2. Validar → 3. Pedir foto ticket → 4. (AI extrae monto o manual) → 5. Confirmar |

#### `internal/modules/earnburn/service.go`

Business logic layer:

```go
type Service struct {
    repo  Repository
    cache Cache
    log   *slog.Logger
}

func (s *Service) AddPoints(ctx context.Context, req AddPointsReq) (*Transaction, error)
func (s *Service) ListPoints(ctx context.Context, clientID, programID uuid.UUID) ([]Transaction, error)
func (s *Service) UpdatePoints(ctx context.Context, req UpdatePointsReq) (*Transaction, error)
func (s *Service) CheckBalance(ctx context.Context, clientID, programID uuid.UUID) (int, error)
func (s *Service) ListRewards(ctx context.Context, customerID, programID uuid.UUID, maxPoints int) ([]Reward, error)
func (s *Service) RequestRedemption(ctx context.Context, req RedemptionReq) (*Redemption, error)
func (s *Service) ConfirmRedemption(ctx context.Context, code string, collaboratorID uuid.UUID) (*Redemption, error)
func (s *Service) RequestLoadPoints(ctx context.Context, clientID uuid.UUID) (string, error)
func (s *Service) ProcessLoadPoints(ctx context.Context, req LoadPointsReq) (*Transaction, error)
```

**Key business rules:**
- `AddPoints`: calculates points from amount using `programs.points_ratio`, stores transaction, updates balance
- `UpdatePoints`: checks `correctable_until > NOW()`, creates adjustment transaction
- `RequestRedemption`: generates 6-char code (crypto/rand), stores via unified OTP system in Redis (1h TTL, type=redemption) + Postgres, deducts points
- `ConfirmRedemption`: validates code from Redis, marks confirmed, records claim ID
- `RequestLoadPoints`: generates temporary code (1h TTL in Redis) for client→collaborator handoff
- `ProcessLoadPoints`: validates client code, processes invoice photo, adds points

#### `internal/modules/earnburn/repository.go`

```go
// Repository interface (enables future extraction to microservice)
type Repository interface {
    GetBalance(ctx context.Context, clientID, programID uuid.UUID) (int, error)
    UpsertBalance(ctx context.Context, clientID, programID uuid.UUID, delta int) (int, error)
    CreateTransaction(ctx context.Context, tx *Transaction) error
    GetTransaction(ctx context.Context, id uuid.UUID) (*Transaction, error)
    ListTransactions(ctx context.Context, clientID, programID uuid.UUID) ([]Transaction, error)
    GetCorrectableTransaction(ctx context.Context, id uuid.UUID) (*Transaction, error)

    ListRewards(ctx context.Context, customerID, programID uuid.UUID, maxPoints int) ([]Reward, error)
    GetReward(ctx context.Context, id uuid.UUID) (*Reward, error)
    CreateReward(ctx context.Context, r *Reward) error
    UpdateReward(ctx context.Context, r *Reward) error

    CreateRedemption(ctx context.Context, r *Redemption) error
    GetRedemptionByCode(ctx context.Context, code string) (*Redemption, error)
    ConfirmRedemption(ctx context.Context, id uuid.UUID, collaboratorID uuid.UUID) error
    ExpirePendingRedemptions(ctx context.Context) (int, error)
}

// PostgresRepository implements Repository
type PostgresRepository struct {
    db *sql.DB
}
```

#### `internal/modules/earnburn/cache.go`

```go
type Cache interface {
    SetRedemptionCode(ctx context.Context, code string, redemptionID uuid.UUID, ttl time.Duration) error
    GetRedemptionCode(ctx context.Context, code string) (uuid.UUID, error)
    DeleteRedemptionCode(ctx context.Context, code string) error

    SetLoadPointsCode(ctx context.Context, code string, clientID uuid.UUID, ttl time.Duration) error
    GetLoadPointsCode(ctx context.Context, code string) (uuid.UUID, error)
    DeleteLoadPointsCode(ctx context.Context, code string) error
}

// RedisCache implements Cache
type RedisCache struct {
    client *redis.Client
}
```

#### `internal/modules/earnburn/api.go`

REST API handlers for admin operations (restructured around programs):

```
POST   /api/v1/customers                              → CreateCustomer
GET    /api/v1/customers/:id                           → GetCustomer
PUT    /api/v1/customers/:id                           → UpdateCustomer
POST   /api/v1/customers/:id/collaborators             → CreateCollaborator
GET    /api/v1/customers/:id/collaborators              → ListCollaborators
POST   /api/v1/programs                                → CreateProgram
GET    /api/v1/programs                                → ListPrograms
PUT    /api/v1/programs/:id                            → UpdateProgram
POST   /api/v1/programs/:program_id/rewards            → CreateReward
GET    /api/v1/programs/:program_id/rewards             → ListRewards
PUT    /api/v1/programs/:program_id/rewards/:id        → UpdateReward
DELETE /api/v1/programs/:program_id/rewards/:id        → DeleteReward (soft)
GET    /api/v1/programs/:program_id/clients/:id/balance       → GetClientBalance
GET    /api/v1/programs/:program_id/clients/:id/transactions  → ListClientTransactions
GET    /api/v1/customers/:id/feedback                  → ListFeedback
GET    /api/v1/customers/:id/registration-info         → GetRegistrationInfo (slug, QR URL)
```

### 7. REST API Layer

#### `api/router.go`
```go
func SetupRouter(bearerToken string, landingHandler, webhookHandler, registry, log *slog.Logger, isDev bool) *gin.Engine {
    r := gin.Default()

    // Landing page (no auth — public)
    r.GET("/unirse/:slug", landingHandler.Join)

    // WhatsApp webhook routes (no auth)
    r.GET("/webhook", webhookHandler.Verify)
    r.POST("/webhook", webhookHandler.Receive)

    // API routes (bearer token auth + error middleware)
    v1 := r.Group("/api/v1")
    v1.Use(middleware.BearerAuth(bearerToken))
    v1.Use(apperror.ErrorHandler(log))  // Converts AppError → JSON with correct HTTP status

    // Customer + collaborator routes
    v1.POST("/customers", ...)
    v1.GET("/customers/:id", ...)
    v1.PUT("/customers/:id", ...)
    v1.POST("/customers/:id/collaborators", ...)
    v1.GET("/customers/:id/collaborators", ...)
    v1.GET("/customers/:id/feedback", ...)
    v1.GET("/customers/:id/registration-info", ...)

    // Program routes
    v1.POST("/programs", ...)
    v1.GET("/programs", ...)
    v1.PUT("/programs/:id", ...)

    // Module routes (each module registers its own under /programs/:program_id/)
    registry.RegisterAllRoutes(v1)

    return r
}
```

#### `api/middleware/auth.go`
```go
func BearerAuth(token string) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if header != "Bearer "+token {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Next()
    }
}
```

---

## Interactive Menu Flow

### Main Menu (per role)

The system presents WhatsApp interactive lists (native UI) based on the user's role. No natural language interpretation — the user selects from predefined options.

**Client menu (5 opciones):**
```
Menu principal — {business_name}:
1. Consultar puntos
2. Ver recompensas (catalogo completo, texto informativo)
3. Canjear recompensa (lista interactiva filtrada por balance)
4. Cargar puntos
5. Dejar feedback
```

**Collaborator menu:**
```
Menu principal — {business_name}:
1. Agregar puntos
2. Consultar puntos de cliente
3. Confirmar canje
4. Corregir transaccion
5. Procesar carga de puntos
```

### Flow Lifecycle (replaces tool_use loop)

```
1. User sends WhatsApp message (or selects menu option)
2. Webhook resolves business context + role
3. Check for active flow in Redis (flow:{phone}:{customer_id})
4. If active flow → process current step (validate input, advance)
5. If menu selection → start new flow (or execute directly for simple queries)
6. If text libre (no active flow, no menu match) → re-present main menu
7. Each flow step: send prompt → wait for response → validate → next step
8. When flow completes → execute business logic → send template response
9. If flow step needs photo → AI Client extracts amount (only AI use)
```

### Session & Flow State

Sessions manage business context resolution. Flow state manages step-by-step interactions. Stored in Redis:

| Key | Value | TTL |
|-----|-------|-----|
| `session:{phone}` | `{customer_id, role, user_id, business_name}` | 30min (reset on each message) |
| `session:select:{phone}` | `[{customer_id, name}, ...]` pending selection | 5min |
| `flow:{phone}:{customer_id}` | `{current_flow, current_step, collected_data, started_at}` | 30min (reset on each interaction) |

**Session resolution flow:**
1. Check `session:{phone}` — if exists, use cached context
2. Extract customer_id from landing page deeplink (data embedded by landing page, no fuzzy match)
3. Lookup phone in `collaborators` + `clients` tables (global index)
4. If found in 1 business → auto-set session
5. If found in multiple businesses → store options in `session:select:{phone}`, present interactive selection list
6. If not found → "Escanea el QR del establecimiento"
7. "Cambiar negocio" menu option → delete `session:{phone}` + `flow:{phone}:*`, re-present resolution

### Edge Cases

| Caso | Resolucion |
|------|-----------|
| Usuario escribe texto libre (no selecciona menu) | Re-presentar menu principal |
| Sesion expirada | Re-presentar menu principal; si no hay sesion, resolver negocio |
| Foto sin sesion activa | Pedir contexto: "Escanea el QR del establecimiento" |
| Foto con sesion pero sin flujo activo | Responder: "Para que necesitas enviar esta foto? Selecciona una opcion del menu" |
| Foto dentro de flujo en paso que espera foto | AI procesa la foto (OCR ticket) |
| Texto pre-llenado del deeplink modificado | Landing page ya capturo customer_id en el deeplink; el texto es solo trigger |
| Cambiar de negocio | Opcion en menu → limpia sesion + flow state |

---

## Hybrid TTL Strategy

| Data | Redis | Postgres | TTL |
|------|-------|----------|-----|
| Session (business context) | `session:{phone}` → `{customer_id, role, user_id, business_name}` | — | 30min |
| Session selection | `session:select:{phone}` → options[] | — | 5min |
| OTP identity | `otp:{code}` → `{client_id, customer_id, type: "identity", metadata: {}}` | — | 15min |
| OTP redemption | `otp:{code}` → `{client_id, customer_id, type: "redemption", metadata: {reward_id, points_spent}}` | `redemptions.expires_at` | 1h |
| OTP load_points | `otp:{code}` → `{client_id, customer_id, type: "load_points", metadata: {}}` | — | 15min |
| Active identity tracker | `otp:active:{client_id}` → code | — | 15min |
| Correction window | — | `transactions.correctable_until` | 2h |
| Flow state | `flow:{phone}:{biz}` → `{current_flow, current_step, collected_data}` | — | 30min |

**Write flow (unified OTP):**
1. Generate 6-char code (crypto/rand)
2. SET `otp:{code}` in Redis with type-specific TTL
3. For type=redemption: also INSERT into Postgres (expires_at = NOW() + 1h)
4. For type=identity: DEL previous OTP via `otp:active:{client_id}`, SET new tracker
5. On confirmation (redemption): GETDEL from Redis, UPDATE Postgres status

**Expiration handling:**
- Redis auto-expires (TTL)
- Postgres: a periodic query or on-access check for `expires_at < NOW()`

---

## Notifications

Proactive WhatsApp messages via background goroutine:

| Event | Message | Trigger |
|-------|---------|---------|
| Points added | "Te han agregado {n} puntos en {negocio}. Balance: {total}" | After add_points |
| Redemption code about to expire | "Tu codigo de canje expira en 30 min: {code}" | 90min after code generation |
| Redemption confirmed | "Canje confirmado: {reward_name}. ID: {claim_id}" | After confirm_redemption |
| New user welcome | "Bienvenido al programa de fidelidad de {negocio}. Tu codigo: {hash}" | First interaction |

**Implementation:** After each operation in the service layer, enqueue notifications. For time-based (30min warning), use a ticker goroutine that checks Redis for codes nearing expiration.

---

## Structured Logging

Using `log/slog` (Go stdlib):

```go
// Every operation logs:
slog.Info("points.added",
    "client_id", clientID,
    "customer_id", customerID,
    "amount", points,
    "balance_after", newBalance,
    "collaborator_id", collabID,
    "duration_ms", elapsed,
)
```

Key logging points:
- WhatsApp message received (phone, business, role)
- Menu selection (command_id, module, role)
- Flow step processed (flow_id, step, duration, success/error)
- AI photo processing (latency, success/error, manual_fallback) — solo para fotos de tickets
- Database operations (operation, table, duration)
- Redemption lifecycle (created, confirmed, expired)

---

## New Dependencies

```
go get github.com/google/uuid                    # UUID generation
go get github.com/minio/minio-go/v7             # S3/MinIO client
```

**Nota sobre Claude API:** Para el procesamiento de fotos de tickets se usa la API directa de Anthropic (HTTP), no un SDK. Es una sola llamada para extraer montos de imagenes — no justifica una dependencia de SDK completa.

Existing deps already cover: Gin, Postgres (lib/pq), Redis (go-redis), godotenv.

---

## Order of Implementation

### Phase 1: Foundation + Migrations
1. `internal/config/config.go` — config struct (with PlatformURL, WhatsAppDisplayPhone)
2. `internal/platform/db/postgres.go` — DB connection
3. `internal/platform/cache/redis.go` — Redis connection
4. `internal/platform/logger/logger.go` — slog setup
5. Migrations 001-004 (platform_config, customers with slug, collaborators, clients)
6. `main.go` — wire config, DB, Redis, logger

### Phase 2: Landing Page + Deeplinks
7. `internal/landing/handler.go` — `GET /unirse/:slug` handler
8. `internal/landing/templates/join.html` — mobile-first landing page
9. `internal/landing/templates/404.html` — business not found
10. `internal/deeplink/generator.go` — wa.me URL generator

### Phase 3: Session + Resolvers
11. `internal/session/manager.go` — Redis session management
12. `internal/resolver/business.go` — resolve business context
13. `internal/resolver/role.go` — resolve role within business

### Phase 4: Module Framework + Flow Engine
14. `internal/loyalty/types.go` — shared types (Command, CommandResult, MenuDefinition, FlowDefinition)
15. `internal/loyalty/module.go` — Module interface (Menus, HandleCommand, FlowDefinitions)
16. `internal/loyalty/registry.go` — registry + command dispatcher
17. `internal/flow/types.go` — FlowState, StepDefinition types
18. `internal/flow/state.go` — Redis persistence of flow state
19. `internal/flow/engine.go` — Flow engine (step management, menu presentation)

### Phase 5: Earn-Burn Module
20. `internal/modules/earnburn/types.go` — module types
21. `internal/modules/earnburn/repository.go` — Repository interface + Postgres impl
22. `internal/modules/earnburn/cache.go` — Cache interface + Redis impl
23. Migrations 005-008 (programs, earnburn tables, rewards, feedback)
24. `internal/modules/earnburn/service.go` — business logic
25. `internal/modules/earnburn/menus.go` — Menu definitions per role + flow step definitions
26. `internal/modules/earnburn/module.go` — wire everything, implement Module

### Phase 6: WhatsApp + Photo Processing
27. `internal/platform/whatsapp/types.go` — payload types (from existing webhook/types.go)
28. `internal/platform/whatsapp/client.go` — send messages (text + interactive lists/buttons)
29. `internal/platform/whatsapp/webhook.go` — receive + resolve business + resolve role + flow engine dispatch
30. `internal/platform/ai/types.go` — photo processing types
31. `internal/platform/ai/client.go` — Claude API client (SOLO para extraer montos de fotos de tickets)

### Phase 7: REST API
32. `api/middleware/auth.go` — bearer token
33. `api/router.go` — API setup (including landing page route)
34. `internal/modules/earnburn/api.go` — earn-burn REST handlers under /programs/:program_id/

### Phase 8: main.go + Wiring + Polish
35. `main.go` — config → DB → Redis → S3 → logger → resolvers → session → flow engine → landing → WhatsApp → modules → router → server
36. Notification system (in-process, goroutine-based)
37. Update `.env.example` with new vars

---

## .env.example Updates

```env
# WhatsApp Business API (single platform number)
WHATSAPP_VERIFY_TOKEN=tu_token_de_verificacion
WHATSAPP_API_TOKEN=tu_token_de_api
WHATSAPP_PHONE_NUMBER_ID=tu_phone_number_id
WHATSAPP_DISPLAY_PHONE=5215551234567

# Platform
PLATFORM_URL=https://fidel.app

# AI — Solo procesamiento de fotos de tickets (OCR)
# No se usa para conversacion — la interaccion es via menus interactivos de WhatsApp
ANTHROPIC_API_KEY=sk-ant-xxxxx

# Database
DATABASE_URL=postgres://loyalty:loyalty@localhost:5433/loyalty?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# S3/MinIO
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=loyalty-invoices
S3_REGION=us-east-1
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin

# Server
PORT=8080
ENV=development

# API Auth
BEARER_TOKEN=dev-token-change-in-production
```

---

## Success Criteria

- [ ] `make start` boots infra, runs migrations, starts app
- [ ] `GET /unirse/cafe-roma` renders landing page with business info + "Unirme por WhatsApp" button
- [ ] Landing page deeplink opens WhatsApp with context embedded
- [ ] WhatsApp verification webhook still works
- [ ] Business resolver correctly identifies business from landing page deeplink data or session
- [ ] Multi-business user gets interactive selection list; session persists for 30min
- [ ] Client receives interactive menu with 5 options upon first message (check_points, list_all_rewards, redeem_rewards, load_points_request, submit_feedback)
- [ ] Collaborator receives interactive menu with 5 options upon first message
- [ ] Menu selections trigger correct step-by-step flows
- [ ] Collaborator can add points via step-by-step flow (OTP → foto/monto → confirmar)
- [ ] Client can check balance via menu selection → direct response
- [ ] Client can request redemption via flow, get code, collaborator confirms via flow
- [ ] AI processes ticket photos and extracts amounts (only AI use in the system)
- [ ] If photo unreadable after 3 attempts, flow falls back to manual amount entry
- [ ] Free text (non-menu) re-presents the main menu
- [ ] REST API CRUD for rewards works under `/api/v1/programs/:program_id/` with bearer token auth
- [ ] Redis TTLs auto-expire redemption codes at 1h
- [ ] Structured JSON logs in production mode
- [ ] Adding a new module requires ONLY: new package + register in main.go

---

## Rollback

All changes are new files except main.go (rewritten) and .env.example (updated).

```bash
# Restore original files
git checkout HEAD -- main.go webhook/handler.go webhook/types.go .env.example

# Remove new files
rm -rf internal/ api/ migrations/

# Remove new deps
go mod tidy
```

---

## Risk Assessment

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|------------|--------|------------|
| 1 | AI photo processing latency for ticket OCR (>5s) | M | L | Send "procesando foto..." indicator; 15s timeout; fallback a ingreso manual despues de 3 intentos |
| 2 | Redis TTL race condition: code expires between check and confirm | L | M | Use Redis GETDEL for atomic check+consume; Postgres as source of truth |
| 3 | Points balance drift (balance != SUM of transactions) | L | H | Wrap balance update + transaction insert in single Postgres TX |
| 4 | WhatsApp rate limits hit during notifications | M | M | Queue notifications; respect Meta rate limits; exponential backoff |
| 5 | WhatsApp interactive message API limitations (max options per list) | L | L | Design menus within WhatsApp limits (max 10 items per section, 3 sections per list) |

**Unmitigated HIGH risks:** 0 (all mitigated)

**Riesgos eliminados vs version anterior (Claude AI conversacional):**
- ~~Claude AI latency en cada mensaje~~ → Respuestas son templates, sin AI (excepto fotos)
- ~~SDK de Anthropic no soporta tool_use~~ → No se usa SDK; API directa solo para fotos
