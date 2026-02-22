# Earn-Burn: Historias de Usuario

> Referencia: `1-earn-burn.md` (requisitos), `2-diagrams.md` (flujos), `earn-burn-proposal.md` (plan tecnico)

---

## Fuera de Scope (futuro)

> Ideas documentadas para referencia futura. NO se implementan en esta version.

| # | Idea | Notas |
|---|------|-------|
| **FS-01** | **Registro via pagina web del establecimiento** | El cliente podria inscribirse al programa de fidelidad desde la web del negocio (no solo via QR/WhatsApp). Requiere integracion con el sitio del customer B2B — fuera de scope actual. |

---

## Reglas de Negocio Invariantes

> Estas reglas se aplican a TODAS las operaciones del sistema. Nunca pueden ser violadas.

| # | Regla | Enforcement |
|---|-------|-------------|
| **RN-01** | **El balance de puntos de un cliente JAMAS puede ser menor a 0** | CHECK constraint en Postgres (`balance >= 0`) + validacion en Service layer antes de cualquier operacion que reste puntos |

**RN-01 aplica en:**
- **CL-04** (redencion): no se puede redencionar si `points_cost > balance`
- **CO-03** (correccion): un ajuste negativo no puede dejar balance < 0
- **SYS-03** (expiracion de redencion): al devolver puntos de un redencion expirado, el balance sube (nunca baja)
- **DB**: `points_balances.balance` tiene `CHECK (balance >= 0)` — ultima linea de defensa

---

## Actores

| Actor | Descripcion | Canal |
|-------|-------------|-------|
| **Cliente** | Usuario final del establecimiento. Acumula y redenciona puntos. | WhatsApp (numero unico de plataforma) |
| **Colaborador** | Empleado del negocio. Opera el sistema de puntos. | WhatsApp (numero unico de plataforma) |
| **Admin** | Dueño del negocio (customer B2B). Gestiona configuracion. | API REST |

---

## Cliente

### CL-01: Registro via QR → Landing page → WhatsApp
**Como** cliente nuevo,
**quiero** escanear el QR del establecimiento, ver la pagina del programa de fidelidad, y unirme por WhatsApp,
**para** quedar inscrito automaticamente y recibir mi primer codigo temporal.

**Criterios de aceptacion:**
- [ ] El registro es 100% self-service via QR
- [ ] El QR apunta a la **landing page** del negocio: `{PLATFORM_URL}/unirse/{slug}` (ej: `https://fidel.app/unirse/cafe-roma`)
- [ ] La landing page (mobile-first) muestra: **nombre del negocio**, **logo**, **descripcion del programa** y boton **"Unirme por WhatsApp"**
- [ ] El boton genera un deeplink: `wa.me/{WHATSAPP_DISPLAY_PHONE}?text=unirme:{customer_id}` — la landing page embebe el customer_id
- [ ] El cliente presiona enviar en WhatsApp — el texto contiene el contexto del negocio
- [ ] El **business resolver** extrae el customer_id directamente del texto del deeplink (no usa fuzzy match ni AI)
- [ ] El sistema registra automaticamente al cliente con su telefono + customer_id del negocio resuelto
- [ ] Se genera un **OTP de 6 caracteres alfanumericos** (crypto/rand) como primer codigo temporal
- [ ] El cliente recibe por WhatsApp: bienvenida con nombre del negocio + su OTP temporal
- [ ] Se indica que el codigo es temporal (15min), que lo use para identificarse con el colaborador, y que puede pedir uno nuevo
- [ ] Se crea automaticamente un registro en `points_balances` con balance 0
- [ ] Se establece sesion en Redis: `session:{phone}` → `{customer_id, role: "client", ...}` TTL 30min
- [ ] Si el cliente ya esta registrado en ese negocio y escribe de nuevo, se le saluda normalmente (no re-registro)
- [ ] Si alguien escribe sin haber escaneado el QR y no tiene sesion ni registros, recibe mensaje indicando que debe escanear el QR del establecimiento
- [ ] Si el slug no existe, la landing page muestra un 404 amigable
- [ ] Si el cliente modifica el texto pre-llenado, la landing page ya capturo el customer_id — el texto es solo trigger

**Ref:** Diagrama 12 (Primer Ingreso), `po_wpp.md`, `1-earn-burn.md` L55, L77

---

### CL-07: Solicitar codigo de identificacion (OTP)
> **Nota:** Esta opcion fue **removida del menu principal** del cliente. El OTP de identidad se genera automaticamente al registrarse (CL-01) y el cliente puede solicitar "Cargar puntos" (CL-05) que genera un codigo de carga, o el colaborador puede usar el OTP de identidad ya existente. Si se necesita reincorporar en el futuro, la logica sigue implementada en el backend (`request_otp` command).

**Como** cliente registrado,
**quiero** pedir un codigo temporal de identificacion en cualquier momento,
**para** darselo al colaborador y que pueda operar sobre mis puntos, sin compartir mi telefono ni datos personales.

**Criterios de aceptacion:**
- [x] ~~El cliente selecciona "Pedir codigo de identificacion" del menu principal~~ (removido del menu — la funcionalidad existe pero no tiene opcion de menu dedicada)
- [ ] Se genera un OTP de 6 caracteres alfanumericos (crypto/rand)
- [ ] El OTP se almacena en Redis: `otp:{code}` → `{client_id, customer_id, type: "identity", metadata: {}}` con **TTL de 15 minutos**
- [ ] Si el cliente ya tiene un OTP activo, se invalida el anterior y se genera uno nuevo (solo 1 activo a la vez)
- [ ] La respuesta incluye: codigo y que es valido por 15 minutos
- [ ] El OTP puede ser usado multiples veces durante su ventana de 15 minutos (GET, no GETDEL)
- [ ] Esto previene suplantacion: el codigo rota constantemente y no se comparte data personal
- [ ] El sistema ya tiene el telefono del cliente via WhatsApp Business, por eso no necesita compartirlo
- [ ] Rate limiting: max 5 intentos fallidos de validacion de OTP por colaborador por minuto

**Nota importante — que tipo de OTP usa cada operacion del colaborador:**

| Operacion del colaborador | Tipo de OTP | Razon |
|---------------------------|-------------|-------|
| add_points (CO-01) | `identity` | No hay otro codigo previo — el cliente comparte su OTP de identidad |
| list_points (CO-02) | `identity` | Consulta directa, no hay codigo previo |
| update_points (CO-03) | `identity` | Seguridad: requiere OTP de identidad vigente incluso si tiene el transaction_id |
| confirm_redemption (CO-04) | `redemption` | El cliente ya genero un OTP type=redemption que contiene su client_id |
| load_points_process (CO-05) | `load_points` | El cliente ya genero un OTP type=load_points que contiene su client_id |

> Todas las operaciones del colaborador requieren un OTP. La diferencia es que en CO-01/02/03 el cliente debe compartir su OTP de identidad explicitamente, mientras que en CO-04/05 el cliente ya genero un OTP de otro tipo que cumple la misma funcion de identificacion.

**Ref:** `1-earn-burn.md` L34 (codigo-usuario), Entrevista: "OTP rotativo, impersonalizacion, TTL 15min"

---

### CL-08: Guia para agregar puntos
**Como** cliente,
**quiero** ver la opcion "agregar puntos" en mis opciones disponibles,
**para** saber como funciona el proceso y que necesito para acreditar mis puntos.

**Criterios de aceptacion:**
- [ ] "Agregar puntos" no aparece como opcion en el menu del cliente — es una operacion exclusiva del colaborador
- [ ] En su lugar, el menu del cliente incluye "Cargar puntos" (CL-05) que guia al cliente a generar un codigo para el colaborador
- [ ] Si el cliente necesita informacion sobre como agregar puntos, el flujo de "Cargar puntos" le indica los pasos:
  > "Genera tu codigo y daselo al colaborador junto con tu ticket de compra."
- [ ] El OTP de identidad se genera automaticamente al registrarse (CL-01); no hay opcion de menu dedicada para generarlo (CL-07 removido del menu)

**Ref:** `1-earn-burn.md` L33-36

---

### CL-09: Ver catalogo de recompensas (motivacional)
**Como** cliente,
**quiero** ver todos los premios disponibles del establecimiento con su costo en puntos,
**para** saber que puedo ganar y motivarme a acumular puntos.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Ver recompensas" (`list_all_rewards`) del menu principal
- [ ] Se listan TODAS las recompensas activas del negocio, sin filtrar por balance
- [ ] Se muestra el balance actual del cliente al inicio: "Tienes X puntos."
- [ ] Cada recompensa muestra: nombre y costo en puntos
- [ ] Si la recompensa es alcanzable con el balance actual, se muestra "disponible"
- [ ] Si la recompensa NO es alcanzable, se muestra "te faltan X pts" — esto motiva al usuario a seguir acumulando
- [ ] Al final se muestra: "Sigue acumulando puntos para desbloquear mas premios."
- [ ] Si no hay recompensas configuradas, se indica con mensaje amigable
- [ ] Es ejecucion directa (sin flujo de pasos) — respuesta como texto, NO lista interactiva

**Diferencia con CL-03:** CL-03 filtra solo las que el cliente puede pagar y las presenta como **lista interactiva seleccionable** para iniciar canje. CL-09 muestra el catalogo completo como **texto informativo** con status de cada recompensa.

**Ref:** `1-earn-burn.md` L61-64

---

### CL-10: Dejar feedback del establecimiento
**Como** cliente,
**quiero** dejar un comentario o valoracion sobre el establecimiento,
**para** compartir mi experiencia y ayudar al negocio a mejorar.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Dejar feedback" del menu principal
- [ ] El flujo solicita el comentario al cliente: "Escribe tu comentario sobre {negocio}:"
- [ ] Se almacena el feedback con: client_id, customer_id, message, created_at
- [ ] Se confirma al cliente que su comentario fue recibido
- [ ] El feedback es visible para el admin via API REST (`GET /api/v1/feedback`)
- [ ] No requiere OTP ni codigo — el cliente ya esta identificado por su sesion de WhatsApp

**Ref:** Entrevista: "opcion de dejar feedback del establecimiento"

---

### CL-02: Consultar balance de puntos
**Como** cliente,
**quiero** consultar cuantos puntos tengo seleccionando una opcion del menu,
**para** saber mi balance actual y mis movimientos recientes.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Consultar puntos" del menu principal
- [ ] El sistema ejecuta la consulta directamente (sin flujo de pasos — ejecucion inmediata)
- [ ] La respuesta incluye: balance total + ultimos 5 movimientos con tipo, monto y fecha
- [ ] Si el cliente no tiene puntos, se indica con mensaje amigable
- [ ] La respuesta llega en menos de 10 segundos

**Ref:** Diagrama 6 (Consultar Puntos), `1-earn-burn.md` L57-58

---

### CL-03: Canjear recompensa (lista interactiva filtrada)
**Como** cliente,
**quiero** ver que premios puedo canjear con mis puntos actuales y seleccionar uno directamente,
**para** iniciar el proceso de redencion de forma rapida.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Canjear recompensa" (`redeem_rewards`) del menu principal
- [ ] Se listan solo las recompensas cuyo `points_cost` es menor o igual al balance del cliente
- [ ] Las recompensas se presentan como **lista interactiva de WhatsApp** (no texto plano) — cada una es seleccionable
- [ ] Cada opcion tiene ID `reward:{reward_id}`, titulo = nombre, descripcion = costo en puntos
- [ ] Si no hay recompensas alcanzables, se muestra: "No tienes puntos suficientes para canjear. Tu balance: X puntos. Sigue acumulando para desbloquear premios."
- [ ] Al seleccionar una recompensa, el flow engine detecta el prefijo `reward:` y llama a `startFlowWithData("request_redemption", {reward_id: ...})`
- [ ] Se informa que el codigo de redencion sera valido por 1 hora

**Diferencia con CL-09:** CL-09 es el catalogo completo (todas las recompensas, texto informativo). CL-03 filtra por balance y presenta como lista interactiva para canjear.

**Ref:** Diagrama 4 Fase 1, `1-earn-burn.md` L61-64

---

### CL-04: Solicitar redencion de recompensa
**Como** cliente,
**quiero** seleccionar una recompensa y recibir un codigo de redencion,
**para** presentarlo al colaborador y reclamar mi premio.

**Criterios de aceptacion:**
- [ ] El cliente selecciona la recompensa de la **lista interactiva de WhatsApp** (CL-03) — cada opcion tiene `reward:{id}` como ID
- [ ] El flow engine inicia `request_redemption` con `reward_id` pre-cargado via `startFlowWithData`
- [ ] El flujo solo tiene 1 paso: confirmacion "Confirmas el canje? (Si/No)"
- [ ] Al confirmar, se genera un codigo alfanumerico de 6 caracteres (crypto/rand)
- [ ] Los puntos se descuentan inmediatamente del balance
- [ ] El codigo se almacena en Redis via sistema OTP unificado: `otp:{code}` → `{client_id, customer_id, type: "redemption", metadata: {reward_id, points_spent}}` con TTL 1h. Tambien en Postgres (expires_at) para auditoria
- [ ] La respuesta incluye: codigo, nombre del premio, tiempo de validez
- [ ] **RN-01**: Si el cliente no tiene suficientes puntos (`points_cost > balance`), se rechaza con mensaje claro. El balance NUNCA puede quedar en negativo.

**Ref:** Diagrama 4 Fase 1, `1-earn-burn.md` L65-68

---

### CL-05: Solicitar carga de puntos
**Como** cliente,
**quiero** generar un codigo temporal de carga,
**para** darselo al colaborador junto con mi ticket y que me acredite los puntos.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Cargar puntos" del menu principal
- [ ] Se genera un codigo alfanumerico de 6 caracteres
- [ ] El codigo se almacena en Redis via sistema OTP unificado: `otp:{code}` → `{client_id, customer_id, type: "load_points", metadata: {}}` con TTL 15min
- [ ] La respuesta incluye: codigo y tiempo de validez (15 minutos)
- [ ] Se indica que debe entregar el codigo + ticket al colaborador
- [ ] Si ya tiene un codigo activo sin usar, se le informa (no generar duplicados)

**Ref:** Diagrama 5 Fase 1, `1-earn-burn.md` L78-81

---

### CL-06: Recibir notificaciones de puntos
**Como** cliente,
**quiero** recibir notificaciones automaticas cuando me agregan puntos o confirman un redencion,
**para** estar al tanto de mis movimientos sin tener que consultar.

**Criterios de aceptacion:**
- [ ] Recibe notificacion al acreditarse puntos: "Te han agregado {n} puntos en {negocio}. Balance: {total}"
- [ ] Recibe notificacion al confirmar redencion: "Canje confirmado: {premio}. ID: {claim_id}"
- [ ] Recibe aviso 30 min antes de que expire un codigo de redencion activo
- [ ] Las notificaciones son mensajes independientes (no requieren conversacion abierta)

**Ref:** Diagrama 3, 4, Plan Seccion Notifications

---

## Colaborador

### CO-01: Agregar puntos a un cliente
**Como** colaborador,
**quiero** asignar puntos a un cliente basado en su compra,
**para** acreditar los puntos de fidelidad que le corresponden.

**Criterios de aceptacion:**
- [ ] El colaborador indica el **OTP temporal** del cliente (6 chars que el cliente le comparte en persona)
- [ ] Se valida el OTP en Redis → resuelve a client_id. Si es invalido o expirado, se rechaza
- [ ] El flujo solicita **foto del ticket** (paso siguiente): "Envia la foto del ticket de compra"
- [ ] AI procesa la foto para extraer el monto automaticamente (unico uso de AI en el sistema)
- [ ] Si la foto no es legible, el flujo indica el problema y pide otra foto (**max 3 intentos**)
- [ ] Si los 3 intentos fallan, se pasa a **ingreso manual**: el flujo solicita el monto directamente: "No pudimos leer la foto. Escribe el monto de la factura:"
- [ ] Los puntos se calculan automaticamente: `floor(monto / points_ratio)`
- [ ] Se registra la transaccion con `correctable_until = NOW() + 2h`
- [ ] **Flag `manual_entry`**: si el monto se ingreso manualmente (sin foto), la transaccion se marca con `manual_entry = true` en la DB
- [ ] La foto del ticket (si existe) se sube a MinIO/S3 y se guarda la URL en la transaccion
- [ ] Se actualiza el balance del cliente atomicamente (dentro de una TX de Postgres)
- [ ] La respuesta incluye: puntos agregados, nuevo balance, ID de transaccion
- [ ] Se envia notificacion al cliente
- [ ] Se informa que la correccion esta disponible por 2 horas

> **Nota de auditoria:** Los ingresos con `manual_entry = true` son riesgosos. La flag permite calcular el rate de ingresos manuales por colaborador y por establecimiento para deteccion de anomalias en el futuro.

**Ref:** Diagrama 3 (Agregar Puntos), `1-earn-burn.md` L33-36

---

### CO-02: Listar puntos de un cliente
**Como** colaborador,
**quiero** consultar el balance y movimientos de un cliente,
**para** verificar su estado de puntos.

**Criterios de aceptacion:**
- [ ] El colaborador proporciona el **OTP temporal** del cliente
- [ ] Se valida el OTP en Redis → resuelve a client_id. Si es invalido o expirado, se rechaza
- [ ] La respuesta incluye: nombre del cliente, balance total, ultimos 10 movimientos
- [ ] Cada movimiento muestra: tipo (earn/burn/adjustment), monto, fecha
- [ ] Si el cliente no existe, se indica con error claro
- [ ] Se marcan las transacciones que aun estan dentro de la ventana de correccion

**Ref:** Diagrama 8 (Listar Puntos), `1-earn-burn.md` L37-38

---

### CO-03: Corregir puntos (ventana de 2 horas)
**Como** colaborador,
**quiero** corregir una transaccion de puntos reciente,
**para** rectificar errores en la carga dentro de las 2 horas permitidas.

**Criterios de aceptacion:**
- [ ] El colaborador identifica al cliente con un **OTP vigente** (se valida en Redis → client_id)
- [ ] Si el OTP original ya expiro (15min < 2h ventana), el colaborador debe pedirle al cliente que genere un nuevo OTP
- [ ] Esto implica que el cliente debe ser contactable para correcciones tardias (seguridad > comodidad)
- [ ] Solo se muestran transacciones con `correctable_until > NOW()`
- [ ] Se muestra el tiempo restante para correccion de cada transaccion
- [ ] El colaborador indica la transaccion y el nuevo monto
- [ ] El flujo solicita **evidencia del error**: "Envia foto o descripcion del error (ej: foto del ticket correcto vs lo registrado)"
- [ ] Se crea una transaccion de tipo `adjustment` (no se modifica la original)
- [ ] El delta (diferencia) se aplica al balance del cliente
- [ ] **RN-01**: Si el ajuste resultaria en balance < 0, se rechaza. El balance NUNCA puede quedar en negativo.
- [ ] Si la ventana de 2h ya expiro, se rechaza con mensaje claro
- [ ] **Despues de aplicar la correccion**, el flujo pregunta al colaborador: "Que fue lo que paso? Escribe un breve comentario del error:"
- [ ] El comentario del colaborador se almacena en la transaccion de adjustment (`correction_reason`)
- [ ] La evidencia (foto si existe) se sube a MinIO/S3 y se guarda en la transaccion (`correction_evidence_url`)
- [ ] Se notifica al cliente del ajuste

> **Nota de auditoria:** El `correction_reason` y `correction_evidence_url` permiten auditar patrones de error por colaborador y establecimiento. Flujo: ver error → verificar → corregir → explicar que paso.

**Ref:** Diagrama 7 (Correccion), `1-earn-burn.md` L39-46

---

### CO-04: Confirmar redencion de recompensa
**Como** colaborador,
**quiero** validar el codigo de redencion que me presenta el cliente,
**para** confirmar la entrega de la recompensa.

**Criterios de aceptacion:**
- [ ] El colaborador proporciona el codigo de redencion del cliente
- [ ] Se valida el codigo en Redis: `GETDEL otp:{code}`, verifica `type == "redemption"`. Fallback a Postgres
- [ ] Si el codigo es valido: se marca como `confirmed`, se registra `confirmed_by` y `confirmed_at`
- [ ] La respuesta incluye: nombre de la recompensa, ID de reclamo (claim_id)
- [ ] Si el codigo no existe o ya expiro, se indica con error claro
- [ ] Si el codigo ya fue confirmado previamente, se indica que ya fue usado
- [ ] Se notifica al cliente que el redencion fue confirmado

**Ref:** Diagrama 4 Fase 2, `1-earn-burn.md` L69-73

---

### CO-05: Procesar carga de puntos con codigo del cliente
**Como** colaborador,
**quiero** procesar una carga de puntos usando el codigo que me da el cliente,
**para** acreditarle los puntos correspondientes a su compra.

**Criterios de aceptacion:**
- [ ] Se valida el codigo del cliente en Redis: `GETDEL otp:{code}`, verifica `type == "load_points"`. Resuelve client_id del payload
- [ ] Si el codigo es valido, se identifica al cliente automaticamente
- [ ] El flujo solicita **foto del ticket**: "Envia la foto del ticket de compra"
- [ ] AI procesa la foto para extraer el monto automaticamente (unico uso de AI en el sistema)
- [ ] Si la foto no es legible, el flujo indica el problema y pide otra foto (**max 3 intentos**)
- [ ] Si los 3 intentos fallan, se pasa a **ingreso manual**: el flujo solicita el monto directamente: "No pudimos leer la foto. Escribe el monto de la factura:"
- [ ] La foto del ticket (si existe) se sube a MinIO/S3 y se guarda la URL en la transaccion
- [ ] Los puntos se calculan con `floor(monto / points_ratio)`
- [ ] **Flag `manual_entry`**: si el monto se ingreso manualmente (sin foto), la transaccion se marca con `manual_entry = true`
- [ ] Se registra transaccion y actualiza balance atomicamente
- [ ] Si el codigo no existe o expiro, se indica con error claro
- [ ] Se notifica al cliente los puntos acreditados

**Ref:** Diagrama 5 Fase 2, `1-earn-burn.md` L82-85

---

## Admin (Negocio B2B)

### AD-01: Gestionar catalogo de recompensas
**Como** admin del negocio,
**quiero** crear, ver, editar y eliminar recompensas de mi catalogo,
**para** definir los premios que mis clientes pueden redencionar.

**Criterios de aceptacion:**
- [ ] `POST /api/v1/programs/:program_id/rewards` — crea recompensa con: nombre, descripcion, costo en puntos
- [ ] `GET /api/v1/programs/:program_id/rewards` — lista todas las recompensas del programa (activas e inactivas)
- [ ] `PUT /api/v1/programs/:program_id/rewards/:id` — edita nombre, descripcion, costo, estado activo
- [ ] `DELETE /api/v1/programs/:program_id/rewards/:id` — desactiva la recompensa (soft delete: `active = false`)
- [ ] Todas las rutas requieren header `Authorization: Bearer {token}`
- [ ] Respuestas en JSON con codigos HTTP estandar (201, 200, 400, 401, 404)

**Ref:** Plan Seccion API Handlers, `1-earn-burn.md` L29-30

---

### AD-02: Consultar balance y transacciones de un cliente
**Como** admin del negocio,
**quiero** consultar el balance y movimientos de cualquier cliente via API,
**para** verificar el estado de cuentas y resolver disputas.

**Criterios de aceptacion:**
- [ ] `GET /api/v1/programs/:program_id/clients/:id/balance` — retorna balance actual
- [ ] `GET /api/v1/programs/:program_id/clients/:id/transactions` — retorna historial paginado
- [ ] Las transacciones incluyen: tipo, monto, balance_after, fecha, colaborador que opero
- [ ] Requiere autenticacion Bearer token
- [ ] Si el cliente no pertenece al negocio del token, retorna 404

**Ref:** Plan Seccion REST API

---

### AD-03: Gestionar colaboradores via API
**Como** admin del negocio,
**quiero** crear y gestionar colaboradores via API,
**para** controlar quien puede operar el sistema de puntos.

**Criterios de aceptacion:**
- [ ] `POST /api/v1/collaborators` — crea colaborador con: nombre, telefono
- [ ] Se genera automaticamente un `hash_id` unico para el colaborador
- [ ] El colaborador queda vinculado al `customer_id` del negocio
- [ ] El telefono del colaborador debe ser unico dentro del negocio
- [ ] Requiere autenticacion Bearer token

**Ref:** `1-earn-burn.md` L29, `0-business.md` L96-108

---

### AD-04: Generar y gestionar QR de registro
**Como** admin del negocio,
**quiero** obtener el QR/link de registro de mi programa de fidelidad,
**para** que mis clientes puedan escanearlo, ver la landing page, y registrarse self-service.

**Criterios de aceptacion:**
- [ ] `GET /api/v1/customers/:id/registration-info` — retorna la URL de la landing page y slug del negocio
- [ ] La URL de registro es: `{PLATFORM_URL}/unirse/{slug}` (ej: `https://fidel.app/unirse/cafe-roma`)
- [ ] El QR codifica esta URL de landing page (no un deeplink directo a WhatsApp)
- [ ] Al escanear, el cliente ve la **landing page** con info del negocio y boton "Unirme por WhatsApp"
- [ ] El boton genera deeplink: `wa.me/{WHATSAPP_DISPLAY_PHONE}?text=Quiero unirme a {nombre_negocio}`
- [ ] Al enviar el mensaje, el business resolver identifica el negocio y registra al cliente (ver CL-01)
- [ ] El admin NO registra clientes manualmente — el registro es siempre self-service via QR → landing → WhatsApp
- [ ] Requiere autenticacion Bearer token

**Ref:** `po_wpp.md`, `1-earn-burn.md` L55, CL-01

---

## Sistema / Transversales

### SYS-01: Identificacion de usuario y resolucion de negocio
**Como** sistema,
**quiero** identificar automaticamente al usuario por su numero de telefono, resolver a que negocio pertenece, y determinar su rol,
**para** construir el contexto completo (negocio + rol) usando un solo numero de WhatsApp de plataforma.

**Criterios de aceptacion:**
- [ ] Se usa **1 solo numero de WhatsApp** compartido por todos los negocios
- [ ] **Resolucion de negocio** (business resolver) sigue este orden:
  1. Buscar sesion activa en Redis (`session:{phone}`) — si existe, usar contexto cacheado
  2. Extraer customer_id de los datos embebidos en el deeplink de la landing page (no fuzzy match, no parseo de texto)
  3. Lookup global del telefono en `collaborators` y `clients` (indices globales en phone)
  4. Si encontrado en **1 solo negocio** → auto-set sesion
  5. Si encontrado en **multiples negocios** → presentar menu de seleccion interactivo, guardar opciones en `session:select:{phone}` TTL 5min
  6. Si **no encontrado** → responder "Escanea el QR del establecimiento para unirte"
- [ ] **Resolucion de rol** (role resolver) dentro del negocio ya resuelto:
  - Buscar phone en `collaborators` para ese `customer_id` → colaborador
  - Buscar phone en `clients` para ese `customer_id` → cliente
  - Si aparece en ambas tablas, **colaborador tiene prioridad**
- [ ] Se construye `UserContext` con: role, user_id, customer_id, business_name, **active_modules**
- [ ] Al crear sesion, se consultan los **programas activos del negocio** (`GetActiveProgramTypes`) y se almacenan como `active_modules` en la sesion (ej: `["earn_burn"]`, `["cashback"]`, `["earn_burn", "cashback"]`)
- [ ] La sesion se guarda en Redis: `session:{phone}` → `{customer_id, role, user_id, business_name, active_modules}` TTL 30min
- [ ] **Al crear una nueva sesion, se limpia cualquier flow state previo** (`ResetFlow`) para evitar que estados de flujo stale de sesiones anteriores interfieran
- [ ] Si una sesion existente no tiene `active_modules` (sesion creada antes del cambio), se hace **backfill automatico**: se consultan los modulos activos, se guardan en la sesion, y se continua
- [ ] El TTL de sesion se reinicia con cada mensaje
- [ ] Opcion **"Usar otro establecimiento"** en el menu principal → borra sesion + flow state → re-trigger resolucion de negocio
- [ ] Si la sesion expira, se re-ejecuta la resolucion de negocio
- [ ] Mensaje sin texto (ej: imagen) sin sesion activa → pedir contexto de negocio

**Ref:** Diagrama 2 (Procesamiento General), `po_wpp.md`

---

### SYS-02: Interaccion por menus interactivos
**Como** sistema,
**quiero** gestionar la interaccion con los usuarios mediante menus interactivos de WhatsApp y flujos paso a paso,
**para** que la experiencia sea rapida, predecible y no dependa de interpretacion de lenguaje natural.

**Criterios de aceptacion:**
- [ ] Cada rol (cliente/colaborador) tiene un menu principal con opciones predefinidas
- [ ] **El menu solo muestra opciones de los modulos con programas activos** para ese negocio (filtrado via `FilteredMenus`). Ej: si el negocio solo tiene cashback, no muestra opciones de puntos
- [ ] Si `active_modules` esta vacio (sesion legacy), se muestran todos los menus como fallback
- [ ] Al final del menu siempre se agrega la opcion **"Usar otro establecimiento"** (`cambiar_negocio`) para permitir al usuario cambiar de negocio
- [ ] Los menus se presentan usando listas interactivas nativas de WhatsApp (no texto plano)
- [ ] Al seleccionar una opcion, se inicia un flujo paso a paso (si requiere datos) o se ejecuta directamente (si es consulta)
- [ ] El flow state se almacena en Redis (`flow:{phone}:{customer_id}`, TTL 30min)
- [ ] El TTL del flow state se reinicia con cada interaccion del usuario
- [ ] Si el flow state expira, se re-presenta el menu principal
- [ ] Si el usuario escribe texto libre (no selecciona una opcion del menu), se re-presenta el menu principal
- [ ] Cada negocio tiene un `welcome_message` personalizado en la tabla `customers` para la bienvenida
- [ ] AI se usa UNICAMENTE para procesar fotos de tickets (OCR → extraer monto). No para conversacion

**Ref:** Diagrama 2, Plan Seccion "Interactive Menu Flow"

---

### SYS-03: Expiracion automatica de codigos
**Como** sistema,
**quiero** que todos los codigos temporales expiren automaticamente,
**para** mantener la seguridad y evitar uso indebido.

**Criterios de aceptacion:**
- [ ] Todos los codigos usan el sistema OTP unificado: `otp:{code}` → `{client_id, customer_id, type, metadata}`
- [ ] **type=identity** (15min TTL): expira por TTL en Redis. Solo 1 activo por cliente (`otp:active:{client_id}`). Se invalida al generar uno nuevo
- [ ] **type=redemption** (1h TTL): expira en Redis (TTL) y Postgres (expires_at). Al expirar pendiente, los puntos se devuelven al balance
- [ ] **type=load_points** (15min TTL): expira por TTL en Redis
- [ ] Redis expira automaticamente via TTL sobre `otp:{code}`
- [ ] Un proceso periodico en Postgres marca `status = 'expired'` y devuelve puntos (solo redemption)
- [ ] Codigos type=redemption y type=load_points usan GETDEL en Redis para consumo atomico (un solo uso)
- [ ] Codigos type=identity pueden ser leidos multiples veces durante su validez (GET, no GETDEL), pero se invalidan al generar uno nuevo

**Ref:** Diagrama 11 (Hybrid TTL), Plan Seccion "Hybrid TTL Strategy"

---

### SYS-06: Sistema OTP de identidad de clientes
**Como** sistema,
**quiero** gestionar codigos OTP rotativos para la identificacion de clientes,
**para** que los colaboradores puedan operar sin acceso a datos personales del cliente.

**Criterios de aceptacion:**
- [ ] El OTP es de 6 caracteres alfanumericos generados con crypto/rand
- [ ] **TTL: 15 minutos** — se informa al cliente en el mensaje de bienvenida y cada vez que genera uno
- [ ] Se almacena en Redis con dos keys: `otp:{code}` → `client_id` (busqueda por codigo) y `otp:active:{client_id}` → `code` (invalidar anterior)
- [ ] Al generar un nuevo OTP de identidad, se elimina el anterior (`DEL otp:{old_code}`, `DEL otp:active:{client_id}`, luego SET nuevos)
- [ ] **Solo 1 OTP activo por cliente** — el ultimo generado invalida todos los anteriores
- [ ] El OTP se genera en: registro via QR (CL-01), solicitud explicita del cliente (CL-07)
- [ ] El colaborador resuelve OTP → client_id con `GET otp:{code}` y verifica `type == "identity"` (no GETDEL — multi-uso durante la ventana de 15min)
- [ ] **Rate limiting**: max 5 intentos fallidos de validacion de OTP por colaborador por minuto
- [ ] Ningun dato personal (telefono, email, nombre) se comparte entre cliente y colaborador — solo el OTP temporal

**Sistema OTP unificado — todos los codigos comparten la misma infraestructura:**

Todos los codigos temporales usan un unico key pattern en Redis:
```
otp:{code} → {client_id, customer_id, type, metadata}
```

| Codigo | type | Proposito | TTL | Uso | metadata | Generado por |
|--------|------|-----------|-----|-----|----------|--------------|
| OTP identidad | `identity` | Colaborador identifica al cliente | **15 min** | Multi-uso (GET) | `{}` | Cliente (CL-01, CL-07) |
| Codigo redencion | `redemption` | Reclamar recompensa | **1h** | Un solo uso (GETDEL) | `{reward_id, points_spent}` | Cliente (CL-04) |
| Codigo carga | `load_points` | Vincular carga a cliente | **15min** | Un solo uso (GETDEL) | `{}` | Cliente (CL-05) |

**Ventajas del sistema unificado:**
- Una sola funcion de generacion de codigos (crypto/rand, 6 chars)
- Una sola estructura Redis — el `type` diferencia el comportamiento
- Validacion centralizada: lookup por codigo, verificar tipo esperado, aplicar logica (GET vs GETDEL)
- Key auxiliar `otp:active:{client_id}` solo aplica para type=identity (invalidar anterior)

**Ref:** Entrevista: "OTP rotativo, impersonalizacion, sin compartir datos personales, TTL 15min"

---

### SYS-04: Logging estructurado
**Como** operador del sistema,
**quiero** logs estructurados de todas las operaciones,
**para** monitorear, depurar y auditar el comportamiento del sistema.

**Criterios de aceptacion:**
- [ ] En desarrollo: logs en formato texto legible con colores
- [ ] En produccion: logs en formato JSON (parseable por herramientas de logging)
- [ ] Cada operacion de puntos loguea: client_id, customer_id, amount, balance_after, duration_ms
- [ ] Cada mensaje de WhatsApp loguea: phone, customer_id, role, message_type
- [ ] Cada llamada a AI (procesamiento de fotos) loguea: latency_ms, success/error, manual_fallback
- [ ] Cada operacion de redencion loguea: code, status, reward_name, lifecycle event

**Ref:** Plan Seccion "Structured Logging"

---

### SYS-05: Modularidad para futuros sistemas de fidelizacion
**Como** desarrollador,
**quiero** que el sistema earn-burn sea un modulo independiente con una interfaz comun,
**para** agregar nuevos sistemas (cashback, tiers, checkin, gamification) sin modificar el core.

**Criterios de aceptacion:**
- [ ] Existe una interfaz `loyalty.Module` con: `Name()`, `Menus()`, `HandleCommand()`, `FlowDefinitions()`, `RegisterRoutes()`
- [ ] El `Registry` tiene `AllMenus(role)` para obtener menus de todos los modulos, y `FilteredMenus(role, activeModules)` para filtrar por modulos activos del negocio
- [ ] El `Registry` despacha comandos (selecciones de menu) al modulo correcto por command_id
- [ ] Agregar un nuevo modulo requiere SOLO: crear package + implementar interfaz + registrar en main.go
- [ ] No se necesita modificar ningun archivo existente para agregar un modulo nuevo
- [ ] Cada modulo tiene su propio repository, cache, service y API handlers

**Ref:** Plan Seccion "Module Interface", Entrevista: "monolito modularizado"

---

## Matriz de Trazabilidad

| Historia | Diagrama | Operacion (1-earn-burn.md) | Menu / Comando | API REST |
|----------|----------|---------------------------|----------------|----------|
| CL-01 | 12 | registro QR → landing → WhatsApp (L55, L77) | — (auto-registro) | GET /customers/:id/registration-info |
| CL-02 | 6 | check-points (L57) | `check_points` (ejecucion directa) | GET /clients/:id/balance |
| CL-03 | 4 | redimir - listar filtrado (L61-64) | `redeem_rewards` (lista interactiva filtrada por balance) | GET /rewards |
| CL-04 | 4 | redimir - codigo (L65-68) | `request_redemption` (1 paso: confirmar; reward_id pre-cargado via startFlowWithData) | — |
| CL-05 | 5 | carga-puntos (L78-81) | `load_points_request` (ejecucion directa) | — |
| CL-06 | 3, 4, 7 | notificaciones | — | — |
| **CL-07** | **—** | **codigo-usuario (L34)** | **~~`request_otp`~~ (removido del menu; logica existe en backend)** | **—** |
| **CL-08** | **—** | **agrega-puntos info (L33)** | **— (informativo via CL-05)** | **—** |
| **CL-09** | **—** | **catalogo recompensas (L61)** | **`list_all_rewards` (ejecucion directa, texto informativo)** | **GET /rewards** |
| **CL-10** | **—** | **feedback establecimiento** | **`submit_feedback` (flujo: pedir comentario)** | **GET /feedback** |
| CO-01 | 3 | agrega-puntos (L33-36) | `add_points` (flujo: OTP → foto/monto → confirmar) | — |
| CO-02 | 8 | lista-puntos (L37-38) | `list_points` (flujo: OTP → mostrar) | GET /clients/:id/transactions |
| CO-03 | 7 | actualizar-puntos (L39-46) | `update_points` (flujo: OTP → seleccionar → corregir → evidencia) | — |
| CO-04 | 4 | remision-puntos (L49-50) | `confirm_redemption` (flujo: codigo → confirmar) | — |
| CO-05 | 5 | carga-puntos (L82-85) | `load_points_process` (flujo: codigo → foto/monto → confirmar) | — |
| AD-01 | — | crud-sistemas (L30) | — | CRUD /rewards |
| AD-02 | — | — | — | GET /programs/:program_id/clients/:id/* |
| AD-03 | — | crud-operador (L29) | — | POST /collaborators |
| AD-04 | — | registro QR → landing (L55) | — | GET /customers/:id/registration-info |
| SYS-01 | 2 | 1 numero plataforma + resolver + sesion | — | — |
| SYS-02 | 2 | menus interactivos + flow engine | todos los menus/flujos | — |
| SYS-03 | 11 | 2h brecha (L43, L68) + OTP | — | — |
| SYS-04 | — | metricas (L16) | — | — |
| SYS-05 | 9 | sistemas de recomendacion | — | — |
| **SYS-06** | **—** | **codigo-usuario (L34, L38, L41)** | **—** | **—** |
