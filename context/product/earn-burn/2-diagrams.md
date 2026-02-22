# Earn-Burn: Diagramas de Flujo y Componentes

---

## 1. Diagrama General de Componentes

```mermaid
graph TB
    subgraph External["Servicios Externos"]
        WA_API["WhatsApp Business API<br/>(Meta)"]
        CLAUDE["Claude API<br/>(Anthropic)<br/>SOLO procesamiento de fotos"]
    end

    subgraph Clients["Actores"]
        CLIENT["Cliente<br/>(Usuario Final)"]
        COLLAB["Colaborador<br/>(Empleado)"]
        ADMIN["Admin<br/>(Dueño Negocio)"]
    end

    subgraph App["fidel-quick (Go / Gin)"]
        subgraph Entrypoints["Puntos de Entrada"]
            WEBHOOK["WhatsApp Webhook<br/>GET /webhook (verify)<br/>POST /webhook (receive)"]
            REST["REST API<br/>/api/v1/*<br/>Bearer Token Auth"]
        end

        subgraph Core["Core del Sistema"]
            LANDING["Landing Page<br/>GET /unirse/:slug<br/>→ wa.me deeplink"]
            BR["Business Resolver<br/>session → deeplink → DB<br/>→ customer_id"]
            RR["Role Resolver<br/>collaborator priority<br/>→ role"]
            SM["Session Manager<br/>Redis TTL 30min"]
            FLOW_ENGINE["Flow Engine<br/>Menu presentation<br/>Step-by-step flows"]
            AI_PHOTO["AI Client<br/>Solo fotos de tickets<br/>OCR → monto"]
            REGISTRY["Module Registry<br/>menu aggregation<br/>+ command dispatch"]
        end

        subgraph ModuleFramework["Framework de Modulos"]
            MOD_IF["loyalty.Module<br/>Interface"]
            MOD_EB["earnburn.Module"]
            MOD_CB["cashback.Module<br/>(futuro)"]
            MOD_TI["tiers.Module<br/>(futuro)"]
        end

        subgraph EarnBurn["Modulo Earn-Burn"]
            EB_MENUS["Menus + Flows<br/>5 cliente / 5 colaborador"]
            EB_SERVICE["Service<br/>Logica de negocio"]
            EB_REPO["Repository<br/>Interface + Postgres"]
            EB_CACHE["Cache<br/>Interface + Redis"]
            EB_API["API Handlers<br/>→ Service (no SQL directo)"]
        end

        subgraph ErrorHandling["Manejo de Errores"]
            APPERROR["apperror.AppError<br/>NotFound, BadRequest,<br/>Internal, Conflict"]
            ERR_MW["Error Middleware<br/>c.Error() → JSON response"]
        end

        subgraph ResolverData["Resolver Repository"]
            RES_REPO["resolver.Repository<br/>Business/Role queries<br/>Landing + Auto-registro"]
        end

        NOTIFIER["Notifier<br/>Mensajes proactivos<br/>via WhatsApp"]
        LOGGER["Logger (slog)<br/>Structured logging"]
    end

    subgraph Infra["Infraestructura"]
        PG["PostgreSQL 16<br/>customers, collaborators,<br/>clients, points, rewards,<br/>redemptions"]
        REDIS["Redis 7<br/>Sessions: session:{phone} (30min)<br/>OTP unificado: otp:{code}<br/>(identity 15min, redemption 1h, load 15min)<br/>Flow state: flow:{phone}:{biz} (30min)"]
        MINIO["MinIO / S3<br/>Fotos de facturas"]
    end

    CLIENT -->|WhatsApp msg| WA_API
    COLLAB -->|WhatsApp msg| WA_API
    ADMIN -->|HTTP request| REST

    WA_API -->|webhook POST| WEBHOOK
    WEBHOOK --> BR
    BR --> SM
    BR --> RR
    RR --> FLOW_ENGINE
    FLOW_ENGINE -->|present menu / process step| REGISTRY
    REGISTRY --> MOD_EB
    FLOW_ENGINE -->|solo fotos de tickets| AI_PHOTO
    AI_PHOTO -->|extract amount| CLAUDE

    MOD_IF -.-|implements| MOD_EB
    MOD_IF -.-|implements| MOD_CB
    MOD_IF -.-|implements| MOD_TI

    MOD_EB --- EB_MENUS
    MOD_EB --- EB_SERVICE
    EB_SERVICE --> EB_REPO
    EB_SERVICE --> EB_CACHE
    MOD_EB --- EB_API

    REST --> ERR_MW --> EB_API
    EB_API --> EB_SERVICE

    APPERROR -.->|classifies errors| EB_SERVICE
    ERR_MW -.->|converts| APPERROR

    BR --> RES_REPO
    RR --> RES_REPO
    LANDING --> RES_REPO
    RES_REPO --> PG

    EB_REPO --> PG
    EB_CACHE --> REDIS
    EB_SERVICE -->|upload invoice| MINIO

    EB_SERVICE -->|enqueue| NOTIFIER
    NOTIFIER -->|send msg| WA_API

    LOGGER -.->|logs| App
```

---

## 2. Flujo General: Mensaje Entrante

Flujo completo desde que un usuario envia un mensaje hasta que recibe respuesta. Cubre resolucion de contexto, sesion, y procesamiento.

```mermaid
sequenceDiagram
    actor U as Usuario
    participant WA as WhatsApp
    participant WH as Webhook
    participant Resolver as Business + Role Resolver
    participant Session as Session Manager
    participant FE as Flow Engine
    participant Registry as Module Registry
    participant Module as Earn-Burn Module

    U->>WA: Envia mensaje o selecciona opcion
    WA->>WH: POST /webhook

    WH->>Session: Buscar sesion activa (phone)

    alt Sesion activa
        Session-->>WH: Contexto del usuario (negocio, rol)
    else Sin sesion
        WH->>Resolver: Identificar negocio y rol
        alt Deeplink desde landing page
            Resolver-->>WH: Negocio identificado por deeplink
        else Un solo negocio registrado
            Resolver-->>WH: Negocio unico auto-seleccionado
        else Multiples negocios
            Resolver->>WA: Lista interactiva: "En cual negocio?"
            Note over U: Fin — espera seleccion
        else No registrado
            Resolver->>WA: "Escanea el QR del establecimiento"
            Note over U: Fin — sin contexto
        end
        WH->>Resolver: Determinar rol (colaborador tiene prioridad)
        Resolver-->>WH: Rol asignado (cliente o colaborador)
        WH->>Session: Crear sesion con contexto
        WH->>FE: ResetFlow (limpiar estado de flujo anterior)
    end

    WH->>FE: Procesar mensaje con contexto de usuario

    alt Flujo activo en curso
        FE->>FE: Validar input del paso actual
        alt Input valido, ultimo paso
            FE->>Registry: Ejecutar comando con datos recopilados
            Registry->>Module: HandleCommand(datos)
            Module-->>FE: Resultado de la operacion
            FE->>WA: Respuesta + menu principal
        else Input valido, mas pasos
            FE->>WA: Prompt del siguiente paso
        else Input invalido
            FE->>WA: Error + repetir paso actual
        end
    else Seleccion de menu (sin flujo activo)
        alt Comando directo (ej: consultar puntos)
            FE->>Registry: Ejecutar comando
            Registry->>Module: HandleCommand
            Module-->>FE: Resultado
            FE->>WA: Respuesta + menu principal
        else Comando multi-paso (ej: agregar puntos)
            FE->>WA: Prompt del primer paso del flujo
        end
    else Texto libre (sin flujo, sin menu)
        FE->>WA: Menu principal segun rol
    end

    WA->>U: Respuesta
```

---

## 3. Flujo: Agregar Puntos (Colaborador)

```mermaid
sequenceDiagram
    actor C as Colaborador
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant AI as AI (Solo Fotos)
    participant EB as Earn-Burn Service
    participant Storage as Almacenamiento
    participant N as Notifier
    actor CL as Cliente

    C->>WA: Selecciona "Agregar puntos" del menu
    WA->>FE: Comando: agregar puntos

    Note over FE: Paso 1: Identificar cliente

    FE->>WA: "Escribe el codigo OTP del cliente:"
    C->>WA: "ABC123"
    WA->>FE: OTP recibido

    FE->>EB: Validar OTP de identidad
    EB-->>FE: Cliente identificado

    Note over FE: Paso 2: Capturar ticket

    FE->>WA: "Envia la foto del ticket de compra:"
    C->>WA: [Foto del ticket]
    WA->>FE: Imagen recibida

    FE->>AI: Extraer monto de la foto
    AI-->>FE: Monto: $1,500

    FE->>Storage: Guardar foto del ticket

    Note over FE: Paso 3: Calcular y acreditar

    FE->>EB: Agregar puntos (cliente, monto $1,500, foto)
    EB->>EB: Calcular puntos segun ratio del programa
    EB->>EB: Registrar transaccion + actualizar balance
    EB-->>FE: 1 punto agregado. Balance: 15 puntos

    FE->>WA: "Se agrego 1 punto. Balance: 15 puntos. Correccion disponible por 2h."
    WA->>C: Confirmacion + menu principal

    EB->>N: Notificar al cliente
    N->>WA: "Te han agregado 1 punto en Negocio X. Balance: 15"
    WA->>CL: Notificacion
```

---

## 4. Flujo: Canje de Recompensa (Cliente + Colaborador)

Flujo completo en dos fases: el cliente solicita el canje y recibe un codigo, luego el colaborador lo confirma en persona.

```mermaid
sequenceDiagram
    actor CL as Cliente
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant EB as Earn-Burn Service
    participant N as Notifier
    actor CO as Colaborador

    Note over CL,CO: FASE 1: Cliente solicita canje

    CL->>WA: Selecciona "Canjear recompensa" del menu
    WA->>FE: Comando: redeem_rewards

    FE->>EB: Obtener balance y recompensas alcanzables (filtradas por balance)
    EB-->>FE: Balance: 50 pts. Recompensas que puede pagar

    alt No tiene puntos suficientes para ninguna recompensa
        FE->>WA: "No tienes puntos suficientes para canjear. Balance: X pts. Sigue acumulando."
        WA->>CL: Mensaje + menu principal
    else Hay recompensas alcanzables
        FE->>WA: Lista interactiva WhatsApp: "Tienes 50 pts. Selecciona recompensa:"
        WA->>CL: Lista con opciones seleccionables (reward:{id})
    end

    CL->>WA: Selecciona "Postre (30 pts)" de la lista interactiva
    WA->>FE: interactive reply: "reward:{reward_id}"

    Note over FE: Flow engine detecta prefijo "reward:" y<br/>inicia flujo request_redemption con reward_id pre-cargado

    FE->>WA: "Confirmas el canje? (Si/No)"
    CL->>WA: "Si"

    FE->>EB: Solicitar canje (Postre, 30 pts)
    EB->>EB: Generar codigo, descontar puntos, registrar canje pendiente
    EB-->>FE: Codigo: X7K9M2, valido por 1 hora

    FE->>WA: "Tu codigo de canje: X7K9M2. Valido por 1 hora. Muestraselo al colaborador."
    WA->>CL: Codigo + menu principal

    Note over CL,CO: FASE 2: Colaborador confirma canje en persona

    CO->>WA: Selecciona "Confirmar canje" del menu
    WA->>FE: Comando: confirmar canje

    FE->>WA: "Escribe el codigo de canje del cliente:"
    CO->>WA: "X7K9M2"
    WA->>FE: Codigo recibido

    FE->>EB: Confirmar canje (codigo X7K9M2)
    EB->>EB: Validar codigo, marcar como confirmado
    EB-->>FE: Canje confirmado: Postre

    FE->>WA: "Canje confirmado. Entrega: Postre."
    WA->>CO: Confirmacion + menu principal

    EB->>N: Notificar al cliente
    N->>WA: "Canje confirmado: Postre. Balance: 20 pts"
    WA->>CL: Notificacion

    Note over EB: Si no se confirma en 1h: codigo expira y puntos se devuelven
```

---

## 5. Flujo: Carga de Puntos (Cliente + Colaborador)

Flujo completo en dos fases: el cliente genera un codigo temporal y se lo da al colaborador junto con su ticket.

```mermaid
sequenceDiagram
    actor CL as Cliente
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant AI as AI (Solo Fotos)
    participant EB as Earn-Burn Service
    participant Storage as Almacenamiento
    participant N as Notifier
    actor CO as Colaborador

    Note over CL,CO: FASE 1: Cliente genera codigo de carga

    CL->>WA: Selecciona "Cargar puntos" del menu
    WA->>FE: Comando: solicitar carga de puntos

    FE->>EB: Generar codigo de carga
    EB-->>FE: Codigo: P8R3K1, valido por 15 minutos

    FE->>WA: "Tu codigo de carga: P8R3K1. Valido por 15 min. Daselo al colaborador junto con tu ticket."
    WA->>CL: Codigo + menu principal

    Note over CL,CO: FASE 2: Colaborador procesa la carga

    CO->>WA: Selecciona "Procesar carga de puntos" del menu
    WA->>FE: Comando: procesar carga

    FE->>WA: "Escribe el codigo del cliente:"
    CO->>WA: "P8R3K1"
    WA->>FE: Codigo recibido

    FE->>EB: Validar codigo de carga
    EB-->>FE: Cliente identificado

    FE->>WA: "Envia la foto del ticket de compra:"
    CO->>WA: [Foto del ticket]
    WA->>FE: Imagen recibida

    FE->>AI: Extraer monto de la foto
    AI-->>FE: Monto: $3,500

    FE->>Storage: Guardar foto del ticket

    FE->>EB: Acreditar puntos (cliente, monto $3,500, foto)
    EB->>EB: Calcular puntos segun ratio del programa
    EB->>EB: Registrar transaccion + actualizar balance
    EB-->>FE: 3 puntos agregados. Balance: 23

    FE->>WA: "Carga exitosa. 3 puntos agregados. Balance: 23"
    WA->>CO: Confirmacion + menu principal

    EB->>N: Notificar al cliente
    N->>WA: "Te cargaron 3 puntos en Negocio X. Balance: 23 pts"
    WA->>CL: Notificacion
```

---

## 6. Flujo: Consultar Puntos (Cliente)

```mermaid
sequenceDiagram
    actor CL as Cliente
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant EB as Earn-Burn Service

    CL->>WA: Selecciona "Consultar puntos" del menu
    WA->>FE: Comando: consultar puntos

    Note over FE: Ejecucion directa (sin flujo de pasos)

    FE->>EB: Consultar balance y movimientos recientes
    EB-->>FE: Balance: 23 pts + ultimos movimientos

    FE->>WA: "Tienes 23 puntos.<br/><br/>Ultimos movimientos:<br/>+3 pts - Carga (13 Feb)<br/>-30 pts - Canje Postre (12 Feb)<br/>+1 pt - Compra (11 Feb)"
    WA->>CL: Balance y movimientos + menu principal
```

---

## 7. Flujo: Correccion de Puntos (Colaborador, ventana 2h)

```mermaid
sequenceDiagram
    actor CO as Colaborador
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant EB as Earn-Burn Service
    participant N as Notifier
    actor CL as Cliente

    CO->>WA: Selecciona "Corregir transaccion" del menu
    WA->>FE: Comando: corregir transaccion

    Note over FE: Paso 1: Identificar cliente

    FE->>WA: "Escribe el codigo OTP del cliente:"
    CO->>WA: "ABC123"
    WA->>FE: OTP recibido

    FE->>EB: Validar OTP de identidad
    EB-->>FE: Cliente identificado

    Note over FE: Paso 2: Seleccionar transaccion

    FE->>EB: Listar transacciones corregibles del cliente
    EB-->>FE: Transacciones dentro de ventana de 2h

    FE->>WA: Lista: "Selecciona transaccion a corregir:"
    WA->>CO: 1. +3 pts (hace 45min, quedan 1h 15min)

    CO->>WA: Selecciona transaccion
    WA->>FE: Transaccion seleccionada

    Note over FE: Paso 3: Nuevo monto

    FE->>WA: "Escribe el monto correcto en puntos:"
    CO->>WA: "5"
    WA->>FE: Nuevo monto recibido

    Note over FE: Paso 4: Evidencia y comentario

    FE->>WA: "Envia foto o descripcion del error:"
    CO->>WA: "El ticket mostraba 5000 pesos, no 3000"
    WA->>FE: Evidencia recibida

    FE->>WA: "Escribe un breve comentario:"
    CO->>WA: "Monto incorrecto en el sistema"
    WA->>FE: Comentario recibido

    Note over FE: Ejecutar correccion

    FE->>EB: Corregir transaccion (de 3 a 5 puntos, evidencia, comentario)
    EB->>EB: Calcular diferencia, registrar ajuste, actualizar balance
    EB-->>FE: Correccion aplicada. 3 a 5 puntos. Balance: 25

    FE->>WA: "Correccion aplicada. 3 → 5 puntos. Balance: 25"
    WA->>CO: Confirmacion + menu principal

    EB->>N: Notificar al cliente
    N->>WA: "Tus puntos fueron ajustados: +2 pts (correccion). Balance: 25"
    WA->>CL: Notificacion
```

---

## 8. Flujo: Consultar Puntos de Cliente (Colaborador)

```mermaid
sequenceDiagram
    actor CO as Colaborador
    participant WA as WhatsApp
    participant FE as Flow Engine
    participant EB as Earn-Burn Service

    CO->>WA: Selecciona "Consultar puntos de cliente" del menu
    WA->>FE: Comando: consultar puntos de cliente

    Note over FE: Paso 1: Identificar cliente

    FE->>WA: "Escribe el codigo OTP del cliente:"
    CO->>WA: "ABC123"
    WA->>FE: OTP recibido

    FE->>EB: Validar OTP de identidad
    EB-->>FE: Cliente identificado

    Note over FE: Paso 2: Mostrar informacion

    FE->>EB: Consultar balance e historial del cliente
    EB-->>FE: Juan Perez, Balance: 25 pts + historial

    FE->>WA: "Cliente: Juan Perez<br/>Balance: 25 puntos<br/><br/>Historial:<br/>+2 pts - Correccion (13 Feb)<br/>+3 pts - Carga (13 Feb)<br/>-30 pts - Canje (12 Feb)"
    WA->>CO: Info del cliente + menu principal
```

---

## 9. Diagrama de Componentes Internos (Capas)

```mermaid
graph LR
    subgraph Entry["Capa de Entrada"]
        LANDING["Landing Page<br/>GET /unirse/:slug"]
        WH["Webhook Handler"]
        API["REST API"]
    end

    subgraph Context["Resolucion de Contexto"]
        BR["Business Resolver<br/>(session → deeplink → DB lookup)"]
        RR["Role Resolver<br/>(collaborator priority)"]
        SM["Session Manager<br/>(Redis TTL 30min)"]
    end

    subgraph Auth["Autenticacion + Errores"]
        BEARER["Bearer Token<br/>(API auth)"]
        ERR_MW2["apperror.ErrorHandler<br/>c.Error() → JSON"]
    end

    subgraph Flows["Capa de Flujos"]
        FE["Flow Engine<br/>(menus + step-by-step flows<br/>+ reward: prefix routing<br/>+ startFlowWithData)"]
        AI_P["AI Client<br/>(solo fotos de tickets)"]
        REG["Module Registry<br/>(command dispatch)"]
    end

    subgraph Business["Capa de Negocio"]
        MOD["loyalty.Module<br/>Interface"]
        SVC["earnburn.Service<br/>(clasifica errores con apperror)"]
    end

    subgraph Data["Capa de Datos"]
        REPO["earnburn.Repository<br/>Interface"]
        CACHE["earnburn.Cache<br/>Interface"]
        RES_REPO2["resolver.Repository<br/>Interface"]
    end

    subgraph Storage["Almacenamiento"]
        PG[(PostgreSQL)]
        RD[(Redis)]
        S3[(MinIO/S3)]
    end

    LANDING -->|deeplink wa.me| WH
    WH --> BR --> SM
    BR --> RR
    SM --> RD
    RR --> FE

    API --> BEARER --> ERR_MW2 --> SVC

    FE --> REG --> MOD --> SVC
    FE -.->|solo fotos| AI_P
    SVC --> REPO --> PG
    SVC --> CACHE --> RD
    SVC --> S3
    BR --> RES_REPO2 --> PG
    RR --> RES_REPO2
```

---

## 10. Diagrama de Datos (ER Simplificado)

```mermaid
erDiagram
    customers ||--o{ collaborators : "emplea"
    customers ||--o{ clients : "tiene clientes"
    customers ||--o{ programs : "configura"
    customers ||--o{ feedback : "recibe feedback"

    programs ||--o{ points_balances : "balance por programa"
    programs ||--o{ points_transactions : "transacciones"
    programs ||--o{ rewards : "premios por programa"
    programs ||--o{ redemptions : "canjes"

    clients ||--o{ points_balances : "acumula"
    clients ||--o{ points_transactions : "registra"
    clients ||--o{ redemptions : "solicita canje"
    clients ||--o{ feedback : "deja feedback"

    collaborators ||--o{ points_transactions : "opera"
    collaborators ||--o{ redemptions : "confirma"

    rewards ||--o{ redemptions : "se canjea por"

    customers {
        uuid id PK
        string name
        string slug UK
        string phone
        text logo_url
        text description
        text welcome_message
        boolean active
    }

    collaborators {
        uuid id PK
        uuid customer_id FK
        string name
        string phone
        string hash_id UK
        boolean active
    }

    clients {
        uuid id PK
        uuid customer_id FK
        string name
        string phone
        string hash UK
    }

    programs {
        uuid id PK
        uuid customer_id FK
        string type
        string name
        int points_ratio
        boolean active
    }

    points_balances {
        uuid id PK
        uuid client_id FK
        uuid program_id FK
        int balance
    }

    points_transactions {
        uuid id PK
        uuid client_id FK
        uuid program_id FK
        uuid collaborator_id FK
        string type
        int amount
        int balance_after
        text invoice_url
        boolean manual_entry
        text correction_reason
        text correction_evidence_url
        timestamp correctable_until
    }

    rewards {
        uuid id PK
        uuid customer_id FK
        uuid program_id FK
        string name
        int points_cost
        boolean active
    }

    redemptions {
        uuid id PK
        uuid client_id FK
        uuid reward_id FK
        uuid program_id FK
        string code UK
        string status
        int points_spent
        uuid confirmed_by FK
        timestamp expires_at
    }

    feedback {
        uuid id PK
        uuid client_id FK
        uuid customer_id FK
        text message
        timestamp created_at
    }
```

---

## 11. Estrategia Hybrid TTL — Sistema OTP Unificado (Redis + Postgres)

```mermaid
graph TB
    subgraph Unified["Sistema OTP Unificado"]
        U0["Todos los codigos: otp:{code}<br/>→ {client_id, customer_id, type, metadata}"]
        U1["type=identity → TTL 15min, GET (multi-uso)"]
        U2["type=redemption → TTL 1h, GETDEL (un solo uso)"]
        U3["type=load_points → TTL 15min, GETDEL (un solo uso)"]
        U0 --- U1
        U0 --- U2
        U0 --- U3
    end

    subgraph Write["Escritura (Generacion de codigo)"]
        W1["1. Generar codigo 6 chars<br/>crypto/rand"]
        W2["2. SET Redis otp:{code}<br/>{client_id, customer_id, type, metadata}<br/>TTL segun type"]
        W3["3. Solo type=redemption:<br/>INSERT Postgres redemptions<br/>expires_at = NOW() + 1h"]
        W1 --> W2 --> W3
    end

    subgraph ReadIdentity["Lectura: type=identity"]
        RI1["GET otp:{code}"]
        RI2{Existe y type==identity?}
        RI3["→ client_id resuelto"]
        RI4["Invalido o expirado"]
        RI1 --> RI2
        RI2 -->|Si| RI3
        RI2 -->|No| RI4
    end

    subgraph ReadSingleUse["Lectura: type=redemption|load_points"]
        RS1["GETDEL otp:{code}"]
        RS2{Existe y type correcto?}
        RS3["Codigo valido<br/>→ Procesar"]
        RS4["Fallback Postgres<br/>(solo redemption)"]
        RS5{Existe y no expirado?}
        RS6["Codigo valido<br/>→ Procesar"]
        RS7["Codigo invalido<br/>o expirado"]
        RS1 --> RS2
        RS2 -->|Si| RS3
        RS2 -->|No| RS4
        RS4 --> RS5
        RS5 -->|Si| RS6
        RS5 -->|No| RS7
    end

    subgraph Expiration["Expiracion"]
        E1["Redis: auto-expire<br/>via TTL sobre otp:{code}"]
        E2["Postgres: periodic job<br/>UPDATE status = 'expired'<br/>WHERE expires_at < NOW()<br/>AND status = 'pending'"]
        E3["Devolver puntos<br/>al balance del cliente<br/>(solo type=redemption)"]
        E2 --> E3
    end

    subgraph Identity["Invalidacion type=identity"]
        ID1["otp:active:{client_id} → code"]
        ID2["Al generar nuevo:<br/>DEL otp:{old_code}<br/>DEL otp:active:{client_id}<br/>SET nuevos keys"]
        ID1 --> ID2
    end
```

---

## 12. Flujo: Registro y Primer Contacto

Flujo unificado con los 4 escenarios posibles cuando un usuario contacta al sistema.

```mermaid
sequenceDiagram
    actor U as Usuario
    participant QR as QR / Landing
    participant WA as WhatsApp
    participant WH as Webhook
    participant Resolver as Resolver
    participant Session as Session Manager
    participant EB as Earn-Burn Service
    participant FE as Flow Engine

    alt Usuario nuevo (deeplink desde QR/landing)
        U->>QR: Escanea QR del establecimiento
        QR-->>U: Landing page del negocio + boton "Unirme por WhatsApp"
        U->>WA: Click en deeplink (envia mensaje con contexto del negocio)
        WA->>WH: Mensaje con deeplink

        WH->>Resolver: Identificar negocio desde deeplink
        Resolver-->>WH: Negocio encontrado (ej: Cafe Roma)

        WH->>Resolver: Buscar usuario en ese negocio
        Resolver-->>WH: No existe — usuario nuevo

        WH->>EB: Registrar cliente nuevo (phone, negocio)
        EB-->>WH: Cliente creado + balance inicial en 0
        WH->>EB: Generar OTP de identidad
        EB-->>WH: Codigo OTP generado

        WH->>Session: Crear sesion (negocio, rol: cliente)

        WH->>WA: Bienvenida + OTP + menu principal de cliente
        WA->>U: "Bienvenido a Cafe Roma! Tu codigo: A7K3M2 (15 min)"

    else Sesion activa (usuario existente)
        U->>WA: Cualquier mensaje
        WA->>WH: Mensaje

        WH->>Session: Buscar sesion activa
        Session-->>WH: Contexto encontrado (negocio, rol)

        FE->>WA: Menu principal segun rol
        WA->>U: Menu principal

    else Multiples negocios (sin sesion)
        U->>WA: Cualquier mensaje (sesion expirada)
        WA->>WH: Mensaje

        WH->>Session: Buscar sesion activa
        Session-->>WH: Sin sesion

        WH->>Resolver: Buscar negocios del usuario
        Resolver-->>WH: Multiples negocios encontrados

        WH->>WA: Lista interactiva: "En cual negocio quieres operar?"
        WA->>U: 1. Cafe Roma / 2. Pizza Lab

        U->>WA: Selecciona "Cafe Roma"
        WA->>WH: Seleccion recibida

        WH->>Resolver: Determinar rol en Cafe Roma
        Resolver-->>WH: Rol: cliente

        WH->>Session: Crear sesion (Cafe Roma, cliente)

        FE->>WA: "Estas en Cafe Roma." + menu principal
        WA->>U: Confirmacion + menu

    else No registrado (sin QR)
        U->>WA: Cualquier mensaje
        WA->>WH: Mensaje

        WH->>Session: Buscar sesion activa
        Session-->>WH: Sin sesion

        WH->>Resolver: Buscar negocios del usuario
        Resolver-->>WH: No encontrado en ningun negocio

        WH->>WA: "Hola! Aun no estas registrado. Escanea el codigo QR en el establecimiento para unirte."
        WA->>U: Mensaje informativo
    end
```
