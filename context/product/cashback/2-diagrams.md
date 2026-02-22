# Cashback: Diagramas de Flujo y Componentes

---

## 1. Diagrama de Componentes del Modulo

```mermaid
graph TB
    subgraph CashbackModule["Modulo Cashback (aislado)"]
        CB_MENUS["Menus + Flows<br/>5 cliente / 5 colaborador"]
        CB_SERVICE["Service<br/>Logica de negocio cashback"]
        CB_REPO["Repository<br/>Interface + Postgres"]
        CB_CACHE["Cache<br/>Interface + Redis"]
        CB_API["API Handlers<br/>→ Service (no SQL directo)"]
    end

    subgraph Shared["Infraestructura Compartida"]
        REGISTRY["Module Registry<br/>(despacha a cashback o earn-burn)"]
        FLOW_ENGINE["Flow Engine<br/>(mismo engine, diferentes flows)"]
        APPERROR["apperror<br/>(mismo manejo de errores)"]
        SESSION["Session Manager<br/>(misma sesion)"]
        WA_CLIENT["WhatsApp Client<br/>(mismo cliente)"]
        RES_REPO["resolver.Repository<br/>(mismos resolvers)"]
    end

    subgraph Storage["Almacenamiento"]
        PG[(PostgreSQL<br/>tablas cashback_*<br/>separadas de earn-burn)]
        REDIS[(Redis<br/>mismos patrones OTP<br/>otp:{code})]
        S3[(MinIO/S3<br/>mismas fotos)]
    end

    REGISTRY --> CB_MENUS
    FLOW_ENGINE --> REGISTRY
    CB_MENUS --> CB_SERVICE
    CB_SERVICE --> CB_REPO --> PG
    CB_SERVICE --> CB_CACHE --> REDIS
    CB_SERVICE --> S3
    CB_API --> CB_SERVICE
    APPERROR -.->|clasifica errores| CB_SERVICE
```

**Nota:** El modulo cashback es completamente independiente de earn-burn. Comparten infraestructura (WhatsApp, Redis, resolvers, flow engine) pero tienen:
- Tablas de DB separadas (`cashback_balances`, `cashback_transactions`, `cashback_rewards`, `cashback_redemptions`)
- Service, Repository y Cache propios
- Menus y flows propios registrados en el Registry
- API handlers propios

---

## 2. Flujo: Acreditar Cashback (Colaborador)

```mermaid
sequenceDiagram
    actor C as Colaborador
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant AI as AI (Solo Fotos)
    participant CB as Cashback Service
    participant Storage as Almacenamiento
    participant N as Notifier
    actor CL as Cliente

    C->>WA: Selecciona "Acreditar cashback" del menu
    WA->>FE: Comando: acreditar cashback

    Note over FE: Paso 1: Identificar cliente

    FE->>WA: "Escribe el codigo OTP del cliente:"
    C->>WA: "ABC123"
    WA->>FE: OTP recibido

    FE->>CB: Validar OTP de identidad
    CB-->>FE: Cliente identificado

    Note over FE: Paso 2: Capturar ticket

    FE->>WA: "Envia la foto del ticket de compra:"
    C->>WA: [Foto del ticket]
    WA->>FE: Imagen recibida

    FE->>AI: Extraer monto de la foto
    AI-->>FE: Monto: $2,000

    FE->>Storage: Guardar foto del ticket

    Note over FE: Paso 3: Calcular y acreditar cashback

    FE->>CB: Acreditar cashback (cliente, monto $2,000, foto)
    CB->>CB: Calcular cashback segun rate del programa (5%)
    CB->>CB: $2,000 * 5% = $100 MXN cashback
    CB->>CB: Registrar transaccion + actualizar saldo
    CB-->>FE: $100 MXN acreditados. Saldo: $350 MXN

    FE->>WA: "Se acredito *$100 MXN* de cashback. Saldo: $350 MXN. Correccion disponible por 2h."
    WA->>C: Confirmacion + menu principal

    CB->>N: Notificar al cliente
    N->>WA: "Te han acreditado $100 MXN de cashback en Negocio X. Saldo: $350"
    WA->>CL: Notificacion
```

---

## 3. Flujo: Canje de Beneficio (Cliente + Colaborador)

```mermaid
sequenceDiagram
    actor CL as Cliente
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant CB as Cashback Service
    participant N as Notifier
    actor CO as Colaborador

    Note over CL,CO: FASE 1: Cliente solicita canje

    CL->>WA: Selecciona "Canjear beneficio" del menu
    WA->>FE: Comando: redeem_cashback

    FE->>CB: Obtener saldo y beneficios alcanzables (filtrados por saldo)
    CB-->>FE: Saldo: $350 MXN. Beneficios que puede pagar

    alt No tiene saldo suficiente para ningun beneficio
        FE->>WA: "No tienes saldo suficiente para canjear. Saldo: $X. Sigue acumulando."
        WA->>CL: Mensaje + menu principal
    else Hay beneficios alcanzables
        FE->>WA: Lista interactiva WhatsApp: "Tienes $350 MXN. Selecciona beneficio:"
        WA->>CL: Lista con opciones seleccionables (benefit:{id})
    end

    CL->>WA: Selecciona "Descuento $200" de la lista interactiva
    WA->>FE: interactive reply: "benefit:{benefit_id}"

    Note over FE: Flow engine detecta prefijo "benefit:" y<br/>inicia flujo request_cashback_redemption con benefit_id pre-cargado

    FE->>WA: "Confirmas el canje? (Si/No)"
    CL->>WA: "Si"

    FE->>CB: Solicitar canje (Descuento $200)
    CB->>CB: Generar codigo, descontar saldo, registrar canje pendiente
    CB-->>FE: Codigo: M4K7R2, valido por 1 hora

    FE->>WA: "Tu codigo de canje: *M4K7R2*. Valido por 1 hora. Muestraselo al colaborador."
    WA->>CL: Codigo + menu principal

    Note over CL,CO: FASE 2: Colaborador confirma canje en persona

    CO->>WA: Selecciona "Confirmar canje" del menu
    WA->>FE: Comando: confirmar canje

    FE->>WA: "Escribe el codigo de canje del cliente:"
    CO->>WA: "M4K7R2"
    WA->>FE: Codigo recibido

    FE->>CB: Confirmar canje (codigo M4K7R2)
    CB->>CB: Validar codigo, marcar como confirmado
    CB-->>FE: Canje confirmado: Descuento $200

    FE->>WA: "Canje confirmado. Entrega: Descuento $200."
    WA->>CO: Confirmacion + menu principal

    CB->>N: Notificar al cliente
    N->>WA: "Canje confirmado: Descuento $200. Saldo: $150"
    WA->>CL: Notificacion

    Note over CB: Si no se confirma en 1h: codigo expira y saldo se devuelve
```

---

## 4. Flujo: Carga de Cashback (Cliente + Colaborador)

```mermaid
sequenceDiagram
    actor CL as Cliente
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant AI as AI (Solo Fotos)
    participant CB as Cashback Service
    participant Storage as Almacenamiento
    participant N as Notifier
    actor CO as Colaborador

    Note over CL,CO: FASE 1: Cliente genera codigo de carga

    CL->>WA: Selecciona "Cargar cashback" del menu
    WA->>FE: Comando: solicitar carga de cashback

    FE->>CB: Generar codigo de carga
    CB-->>FE: Codigo: T3N8P5, valido por 15 minutos

    FE->>WA: "Tu codigo de carga: *T3N8P5*. Valido por 15 min. Daselo al colaborador junto con tu ticket."
    WA->>CL: Codigo + menu principal

    Note over CL,CO: FASE 2: Colaborador procesa la carga

    CO->>WA: Selecciona "Procesar carga de cashback" del menu
    WA->>FE: Comando: procesar carga

    FE->>WA: "Escribe el codigo del cliente:"
    CO->>WA: "T3N8P5"
    WA->>FE: Codigo recibido

    FE->>CB: Validar codigo de carga
    CB-->>FE: Cliente identificado

    FE->>WA: "Envia la foto del ticket de compra:"
    CO->>WA: [Foto del ticket]
    WA->>FE: Imagen recibida

    FE->>AI: Extraer monto de la foto
    AI-->>FE: Monto: $5,000

    FE->>Storage: Guardar foto del ticket

    FE->>CB: Acreditar cashback (cliente, monto $5,000, foto)
    CB->>CB: Calcular cashback: $5,000 * 5% = $250
    CB->>CB: Registrar transaccion + actualizar saldo
    CB-->>FE: $250 MXN acreditados. Saldo: $600

    FE->>WA: "Carga exitosa. *$250 MXN* de cashback acreditados. Saldo: $600"
    WA->>CO: Confirmacion + menu principal

    CB->>N: Notificar al cliente
    N->>WA: "Te acreditaron $250 MXN de cashback en Negocio X. Saldo: $600"
    WA->>CL: Notificacion
```

---

## 5. Flujo: Consultar Saldo (Cliente)

```mermaid
sequenceDiagram
    actor CL as Cliente
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant CB as Cashback Service

    CL->>WA: Selecciona "Consultar saldo" del menu
    WA->>FE: Comando: consultar saldo cashback

    Note over FE: Ejecucion directa (sin flujo de pasos)

    FE->>CB: Consultar saldo y movimientos recientes
    CB-->>FE: Saldo: $600 MXN + ultimos movimientos

    FE->>WA: "Tu saldo cashback: *$600 MXN*.<br/><br/>Ultimos movimientos:<br/>+$250 - Carga (13 Feb)<br/>-$200 - Canje Descuento (12 Feb)<br/>+$100 - Carga (11 Feb)"
    WA->>CL: Saldo y movimientos + menu principal
```

---

## 6. Flujo: Correccion de Cashback (Colaborador, ventana 2h)

```mermaid
sequenceDiagram
    actor CO as Colaborador
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant CB as Cashback Service
    participant N as Notifier
    actor CL as Cliente

    CO->>WA: Selecciona "Corregir transaccion" del menu
    WA->>FE: Comando: corregir transaccion

    Note over FE: Paso 1: Identificar cliente

    FE->>WA: "Escribe el codigo OTP del cliente:"
    CO->>WA: "ABC123"
    WA->>FE: OTP recibido

    FE->>CB: Validar OTP de identidad
    CB-->>FE: Cliente identificado

    Note over FE: Paso 2: Seleccionar transaccion

    FE->>CB: Listar transacciones corregibles del cliente
    CB-->>FE: Transacciones dentro de ventana de 2h

    FE->>WA: Lista: "Selecciona transaccion a corregir:"
    WA->>CO: 1. +$250 (hace 45min, quedan 1h 15min)

    CO->>WA: Selecciona transaccion
    WA->>FE: Transaccion seleccionada

    Note over FE: Paso 3: Nuevo monto

    FE->>WA: "Escribe el monto correcto de la factura (en pesos):"
    CO->>WA: "6000"
    WA->>FE: Nuevo monto recibido

    Note over FE: Paso 4: Evidencia y comentario

    FE->>WA: "Envia foto o descripcion del error:"
    CO->>WA: "El ticket mostraba 6000 pesos, no 5000"
    WA->>FE: Evidencia recibida

    FE->>WA: "Escribe un breve comentario:"
    CO->>WA: "Monto incorrecto en el sistema"
    WA->>FE: Comentario recibido

    Note over FE: Ejecutar correccion

    FE->>CB: Corregir transaccion (recalcular cashback con nuevo monto)
    CB->>CB: Recalcular: $6,000 * 5% = $300 (antes $250). Diferencia: +$50
    CB->>CB: Registrar ajuste, actualizar saldo
    CB-->>FE: Correccion aplicada. Ajuste: +$50. Saldo: $650

    FE->>WA: "Correccion aplicada. Ajuste: +$50. Saldo: $650"
    WA->>CO: Confirmacion + menu principal

    CB->>N: Notificar al cliente
    N->>WA: "Tu cashback fue ajustado: +$50 (correccion). Saldo: $650"
    WA->>CL: Notificacion
```

---

## 7. Flujo: Consultar Saldo de Cliente (Colaborador)

```mermaid
sequenceDiagram
    actor CO as Colaborador
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant CB as Cashback Service

    CO->>WA: Selecciona "Consultar saldo de cliente" del menu
    WA->>FE: Comando: consultar saldo de cliente

    Note over FE: Paso 1: Identificar cliente

    FE->>WA: "Escribe el codigo OTP del cliente:"
    CO->>WA: "ABC123"
    WA->>FE: OTP recibido

    FE->>CB: Validar OTP de identidad
    CB-->>FE: Cliente identificado

    Note over FE: Paso 2: Mostrar informacion

    FE->>CB: Consultar saldo e historial del cliente
    CB-->>FE: Juan Perez, Saldo: $650 MXN + historial

    FE->>WA: "Cliente: *Juan Perez*<br/>Saldo cashback: *$650 MXN*<br/><br/>Historial:<br/>+$50 - Correccion (13 Feb)<br/>+$250 - Carga (13 Feb)<br/>-$200 - Canje (12 Feb)"
    WA->>CO: Info del cliente + menu principal
```

---

## 8. Diagrama de Datos (ER Cashback)

```mermaid
erDiagram
    customers ||--o{ cashback_programs : "configura"
    customers ||--o{ cashback_rewards : "define beneficios"

    cashback_programs ||--o{ cashback_balances : "saldo por programa"
    cashback_programs ||--o{ cashback_transactions : "transacciones"
    cashback_programs ||--o{ cashback_rewards : "beneficios por programa"
    cashback_programs ||--o{ cashback_redemptions : "canjes"

    clients ||--o{ cashback_balances : "acumula"
    clients ||--o{ cashback_transactions : "registra"
    clients ||--o{ cashback_redemptions : "solicita canje"

    collaborators ||--o{ cashback_transactions : "opera"
    collaborators ||--o{ cashback_redemptions : "confirma"

    cashback_rewards ||--o{ cashback_redemptions : "se canjea por"

    cashback_programs {
        uuid id PK
        uuid customer_id FK
        string type "cashback"
        string name
        decimal cashback_rate "ej 0.05 = 5%"
        boolean active
    }

    cashback_balances {
        uuid id PK
        uuid client_id FK
        uuid program_id FK
        decimal balance "en pesos MXN"
    }

    cashback_transactions {
        uuid id PK
        uuid client_id FK
        uuid program_id FK
        uuid collaborator_id FK
        string type "earn|burn|adjustment"
        decimal amount "en pesos MXN"
        decimal purchase_amount "monto original de compra"
        decimal balance_after
        text invoice_url
        boolean manual_entry
        text correction_reason
        text correction_evidence_url
        timestamp correctable_until
    }

    cashback_rewards {
        uuid id PK
        uuid customer_id FK
        uuid program_id FK
        string name
        text description
        decimal cost "en pesos MXN"
        boolean active
    }

    cashback_redemptions {
        uuid id PK
        uuid client_id FK
        uuid reward_id FK
        uuid program_id FK
        string code UK
        string status "pending|confirmed|expired|cancelled"
        decimal amount_spent "en pesos MXN"
        uuid confirmed_by FK
        timestamp expires_at
        timestamp confirmed_at
    }
```

**Diferencias clave con earn-burn:**
- `cashback_programs.cashback_rate` (decimal, ej: 0.05) en lugar de `programs.points_ratio` (integer)
- `cashback_balances.balance` es `DECIMAL(12,2)` (pesos) en lugar de `INTEGER` (puntos)
- `cashback_transactions.amount` es `DECIMAL(12,2)` (pesos) en lugar de `INTEGER` (puntos)
- `cashback_transactions.purchase_amount` almacena el monto original de la compra para auditoria
- `cashback_rewards.cost` es `DECIMAL(12,2)` (pesos) en lugar de `INTEGER` (puntos)
- Todas las tablas tienen prefijo `cashback_` para aislamiento total

**Tablas compartidas (no se duplican):**
- `customers` — el negocio B2B
- `clients` — el usuario final
- `collaborators` — los empleados
- `feedback` — ya es por customer_id, sirve para ambos modulos
