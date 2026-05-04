# Plan: Arquitectura Modular — Sistemas de Fidelización (SISFI)

**Mode:** DEEP
**Contexto:** Renombrar "programs" a "sisfi" (sistemas de fidelización). Cada módulo es dueño de su configuración en su propio dominio. Una tabla central `sisfi` registra los tipos disponibles y `customer_sisfi` los vincula a cada negocio.

---

## Arquitectura de Datos

### 3 capas

```
sisfi                    ← catálogo de tipos de sistemas de fidelización
  |
customer_sisfi           ← qué negocios tienen qué sistemas activos
  |
[módulo]_config          ← configuración específica por módulo (dominio propio)
[módulo]_balances
[módulo]_transactions
[módulo]_rewards
[módulo]_redemptions
```

### Capa 1: Catálogo — `sisfi`

Tabla de control con los sistemas de fidelización que administra la plataforma.

```sql
CREATE TABLE sisfi (
    id VARCHAR(50) PRIMARY KEY,       -- 'earn_burn', 'cashback', 'pushcard', 'tiers', 'gamification'
    name VARCHAR(255) NOT NULL,       -- 'Puntos', 'Cashback', 'Tarjeta de Sellos', etc.
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO sisfi (id, name, description) VALUES
('earn_burn', 'Puntos', 'Acumula puntos por compras y canjéalos por recompensas'),
('cashback', 'Cashback', 'Recibe un porcentaje de vuelta por cada compra');
```

**Esto permite:**
- Saber qué sistemas de fidelización existen en la plataforma
- Desactivar un sistema globalmente (`active = false`)
- Mostrar catálogo al customer durante onboarding

### Capa 2: Vinculación — `customer_sisfi`

Qué negocios tienen qué sistemas activos.

```sql
CREATE TABLE customer_sisfi (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    sisfi_id VARCHAR(50) NOT NULL REFERENCES sisfi(id),
    name VARCHAR(255) NOT NULL,       -- nombre personalizado: "Mis Puntazos"
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(customer_id, sisfi_id)     -- 1 por tipo por ahora (removible en el futuro)
);
```

**`UNIQUE(customer_id, sisfi_id)`:** Limita a 1 sistema por tipo por customer. Removible cuando se necesiten múltiples programas del mismo tipo.

### Capa 3: Configuración por módulo (dominio propio)

Cada módulo es dueño de su configuración. Columnas tipadas, constraints a nivel DB.

**earn_burn:**
```sql
CREATE TABLE earn_burn_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL UNIQUE REFERENCES customer_sisfi(id),
    points_ratio INTEGER NOT NULL DEFAULT 1000 CHECK (points_ratio > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**cashback:**
```sql
CREATE TABLE cashback_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL UNIQUE REFERENCES customer_sisfi(id),
    cashback_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0500 CHECK (cashback_rate > 0 AND cashback_rate <= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**pushcard (futuro):**
```sql
CREATE TABLE pushcard_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL UNIQUE REFERENCES customer_sisfi(id),
    card_slots INTEGER NOT NULL DEFAULT 10 CHECK (card_slots > 0),
    reward_on_complete UUID,  -- FK a pushcard_rewards
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Tablas operacionales por módulo

Las tablas de balances, transactions, rewards y redemptions siguen siendo por módulo (independientes), con FK a `customer_sisfi.id`:

| Módulo | Tablas |
|--------|--------|
| earn_burn | `earn_burn_config`, `points_balances`, `points_transactions`, `rewards`, `redemptions` |
| cashback | `cashback_config`, `cashback_balances`, `cashback_transactions`, `cashback_rewards`, `cashback_redemptions` |
| pushcard | `pushcard_config`, `pushcard_cards`, `pushcard_stamps` |
| tiers | `tiers_config`, `tier_levels`, `tier_memberships` |

### Diagrama completo

```
sisfi (catálogo global)
  │
  ├── earn_burn
  ├── cashback
  ├── pushcard
  └── ...

customers
  │
  └── customer_sisfi (vinculación: qué sistemas tiene cada negocio)
        │
        ├── earn_burn_config          ← dominio earn_burn
        │     points_balances
        │     points_transactions
        │     rewards
        │     redemptions
        │
        ├── cashback_config           ← dominio cashback
        │     cashback_balances
        │     cashback_transactions
        │     cashback_rewards
        │     cashback_redemptions
        │
        └── pushcard_config           ← dominio pushcard (futuro)
              pushcard_cards
              pushcard_stamps
```

---

## Cambios en el Backend

### Resolver
```go
// Nombre mantenido por compatibilidad, query actualizada:
func GetActiveProgramTypes(ctx, customerID) ([]string, error)
// Antes: UNION de programs + cashback_programs
// Después: SELECT cs.sisfi_id FROM customer_sisfi cs
//          JOIN sisfi s ON s.id = cs.sisfi_id AND s.active = true
//          WHERE cs.customer_id = $1 AND cs.active = true
```

### Module Interface
```go
type Module interface {
    Name() string                                                          // "earn_burn", "cashback"
    Menus() map[string][]MenuDefinition
    Prefixes() []string                                                    // ["reward:"] o ["cb_reward:"]
    SelectionFlow(prefix string) (commandID string, dataKey string)        // mapea prefix → flow
    HandleCommand(ctx, cmd) (*CommandResult, error)
    FlowDefinitions() map[string]FlowDefinition
    RegisterRoutes(rg *gin.RouterGroup)
}
```

### Registry
```go
type Registry struct {
    modules  map[string]Module
    commands map[string]string    // command_id → module_name
    prefixes map[string]string    // prefix → module_name
}

func (r *Registry) ResolveSelection(selectionID string) (string, map[string]string)
```

### Flow Engine
```go
// engine.go — reemplaza prefixes hardcodeados:
if flowCmd, flowData := e.registry.ResolveSelection(commandID); flowCmd != "" {
    return e.startFlowWithData(ctx, user, flowCmd, flowData)
}
```

---

## Cambios en el Frontend

### Tipos
```typescript
interface Sisfi {
  id: string          // 'earn_burn', 'cashback', ...
  name: string        // 'Puntos', 'Cashback', ...
  description: string
  active: boolean
}

interface CustomerSisfi {
  id: string
  customer_id: string
  sisfi_id: string
  name: string        // nombre personalizado
  active: boolean
}
```

### API
```
GET  /sisfi                          → catálogo disponible
GET  /customer-sisfi?customer_id=X   → sistemas activos del customer
POST /customer-sisfi                 → activar un sistema
PUT  /customer-sisfi/{id}            → actualizar
```

---

## Migración

**No se migran datos existentes.** Las tablas `programs` y `cashback_programs` se eliminan.

### Fase 1: Crear nuevas tablas
1. Crear `sisfi` con datos iniciales (earn_burn, cashback)
2. Crear `customer_sisfi`
3. Crear `earn_burn_config` y `cashback_config`

### Fase 2: Refactorizar FKs
1. Las tablas operacionales existentes (`points_balances`, etc.) cambian su FK de `programs.id` → `customer_sisfi.id`
2. Eliminar tablas `programs` y `cashback_programs`
3. Eliminar CHECK constraints de migración 000015

### Fase 3: Refactorizar código
1. Actualizar resolver: query de `GetActiveProgramTypes()` para leer de `customer_sisfi`
2. Actualizar Module interface: agregar `Prefixes()` + `SelectionFlow()`
3. Actualizar Registry: agregar prefix routing
4. Actualizar engine.go: delegar selecciones via Registry
5. Actualizar frontend: nuevos tipos y endpoints
6. Actualizar tests y mocks

---

## Riesgos

| Riesgo | Severidad | Mitigación |
|--------|-----------|------------|
| FK migration rompe queries | ALTA | Fase 2 cambia FKs con datos frescos (no hay migración de datos legacy) |
| Module interface breaking change | MEDIA | Agregar métodos con defaults: `Prefixes() → []string{}`, `SelectionFlow() → ("", "")` |
| Session stale con ActiveModules viejo | BAJA | Backfill en webhook.go ya existe, solo cambiar la query |

---

## Success Criteria

- [ ] Tabla `sisfi` como catálogo de sistemas disponibles
- [ ] `customer_sisfi` como registro de qué negocios tienen qué sistemas
- [ ] Config de cada módulo en su propia tabla con constraints tipados
- [ ] Agregar un nuevo módulo = crear paquete + config table + implementar interface + registrar
- [ ] NO se modifica resolver, engine, ni tablas de otros módulos
- [ ] Frontend usa `Sisfi` y `CustomerSisfi` en vez de `Program`/`CashbackProgram`

## Test Strategy

- [ ] Test: crear sisfi + customer_sisfi + config, verificar `GetActiveProgramTypes()` lo devuelve
- [ ] Test: cashback_config rechaza rate <= 0 o > 1 (constraint DB)
- [ ] Test: earn_burn_config rechaza points_ratio <= 0 (constraint DB)
- [ ] Test: agregar módulo dummy NO requiere cambios en resolver/engine
- [ ] Test: prefix routing despacha correctamente a cada módulo
- [ ] Test: mocks actualizados en business_test.go

## Rollback

| Fase | Rollback |
|------|----------|
| Fase 1 | DROP nuevas tablas (sisfi, customer_sisfi, *_config) |
| Fase 2 | Re-crear tablas programs/cashback_programs, restaurar FKs |
| Fase 3 | Revertir Module interface, restaurar CutPrefix en engine.go |
