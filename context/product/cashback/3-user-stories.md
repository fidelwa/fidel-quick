# Cashback: Historias de Usuario

> Referencia: `1-cashback.md` (requisitos), `2-diagrams.md` (flujos), `cashback-proposal.md` (plan tecnico)
> Paralelo: Las historias siguen la misma estructura que earn-burn. La diferencia fundamental es que la unidad de balance es **pesos (dinero)**, no puntos.

---

## Fuera de Scope (futuro)

| # | Idea | Notas |
|---|------|-------|
| **FS-01** | **Cashback escalonado por categoria** | Diferentes porcentajes de cashback segun la categoria del producto (ej: 10% en bebidas, 3% en comida). Requiere categorizacion de productos — fuera de scope actual. |
| **FS-02** | **Cashback con tope maximo** | Limite maximo de cashback por transaccion o por periodo. Configuracion adicional — fuera de scope actual. |

---

## Reglas de Negocio Invariantes

| # | Regla | Enforcement |
|---|-------|-------------|
| **RN-01** | **El saldo cashback de un cliente JAMAS puede ser menor a $0** | CHECK constraint en Postgres (`balance >= 0`) + validacion en Service layer |
| **RN-02** | **El cashback se calcula como porcentaje del monto de compra** | `cashback = floor(purchase_amount * cashback_rate * 100) / 100` (redondeo a 2 decimales hacia abajo) |

**RN-01 aplica en:**
- **CB-CL-04** (canje): no se puede canjear si `cost > balance`
- **CB-CO-03** (correccion): un ajuste negativo no puede dejar saldo < 0
- **CB-SYS-01** (expiracion de canje): al devolver saldo de un canje expirado, el balance sube
- **DB**: `cashback_balances.balance` tiene `CHECK (balance >= 0)`

---

## Actores

| Actor | Descripcion | Canal |
|-------|-------------|-------|
| **Cliente** | Usuario final del establecimiento. Acumula y canjea cashback. | WhatsApp (mismo numero de plataforma) |
| **Colaborador** | Empleado del negocio. Opera el sistema de cashback. | WhatsApp (mismo numero de plataforma) |
| **Admin** | Dueno del negocio (customer B2B). Gestiona configuracion. | API REST |

---

## Cliente

### CB-CL-01: Registro (compartido con earn-burn)
El registro del cliente es el mismo proceso que earn-burn (CL-01). Un cliente registrado puede tener programas earn-burn, cashback, o ambos con el mismo negocio. No se requiere registro adicional.

---

### CB-CL-02: Consultar saldo cashback
**Como** cliente,
**quiero** consultar cuanto cashback tengo acumulado,
**para** saber mi saldo actual y mis movimientos recientes.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Consultar saldo" (`cb_check_balance`) del menu principal
- [ ] Ejecucion directa (sin flujo de pasos)
- [ ] La respuesta incluye: saldo total en pesos + ultimos 5 movimientos con tipo, monto y fecha
- [ ] Los montos se formatean con simbolo de moneda: `$100 MXN`
- [ ] Si el cliente no tiene saldo, se indica con mensaje amigable

**Ref:** Diagrama 5 (Consultar Saldo)

---

### CB-CL-03: Canjear beneficio (lista interactiva filtrada)
**Como** cliente,
**quiero** ver que beneficios puedo canjear con mi saldo actual y seleccionar uno,
**para** usar mi cashback acumulado.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Canjear beneficio" (`cb_redeem`) del menu principal
- [ ] Se listan solo los beneficios cuyo `cost` es menor o igual al saldo del cliente
- [ ] Los beneficios se presentan como **lista interactiva de WhatsApp** — cada uno es seleccionable
- [ ] Cada opcion tiene ID `benefit:{reward_id}`, titulo = nombre, descripcion = costo en pesos
- [ ] Si no hay beneficios alcanzables: "No tienes saldo suficiente para canjear. Saldo: $X MXN. Sigue acumulando."
- [ ] Al seleccionar, el flow engine detecta el prefijo `benefit:` y llama a `startFlowWithData("cb_request_redemption", {reward_id: ...})`
- [ ] **RN-01**: Si el saldo es insuficiente, se rechaza. El saldo NUNCA puede quedar en negativo.

**Ref:** Diagrama 3 Fase 1

---

### CB-CL-04: Solicitar canje de beneficio
**Como** cliente,
**quiero** confirmar un canje y recibir un codigo,
**para** presentarlo al colaborador y reclamar mi beneficio.

**Criterios de aceptacion:**
- [ ] El cliente selecciona el beneficio de la lista interactiva (CB-CL-03) — `benefit:{id}`
- [ ] El flow engine inicia `cb_request_redemption` con `reward_id` pre-cargado via `startFlowWithData`
- [ ] El flujo solo tiene 1 paso: confirmacion "Confirmas el canje? (Si/No)"
- [ ] Al confirmar, se genera un codigo alfanumerico de 6 caracteres
- [ ] El saldo se descuenta inmediatamente
- [ ] El codigo se almacena en Redis: `otp:{code}` → `{..., type: "cb_redemption", metadata: {reward_id, amount_spent}}` con TTL 1h. Tambien en Postgres
- [ ] La respuesta incluye: codigo, nombre del beneficio, tiempo de validez

**Ref:** Diagrama 3 Fase 1

---

### CB-CL-05: Solicitar carga de cashback
**Como** cliente,
**quiero** generar un codigo temporal,
**para** darselo al colaborador junto con mi ticket y que me acredite el cashback.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Cargar cashback" (`cb_load_request`) del menu principal
- [ ] Se genera un codigo alfanumerico de 6 caracteres
- [ ] El codigo se almacena en Redis: `otp:{code}` → `{..., type: "cb_load_points"}` con TTL 15min
- [ ] La respuesta incluye: codigo y tiempo de validez (15 minutos)
- [ ] Se indica que debe entregar el codigo + ticket al colaborador

**Ref:** Diagrama 4 Fase 1

---

### CB-CL-06: Recibir notificaciones de cashback
**Como** cliente,
**quiero** recibir notificaciones cuando me acreditan cashback o confirman un canje,
**para** estar al tanto de mis movimientos.

**Criterios de aceptacion:**
- [ ] Notificacion al acreditarse cashback: "Te han acreditado $X MXN de cashback en {negocio}. Saldo: $Y"
- [ ] Notificacion al confirmar canje: "Canje confirmado: {beneficio}. Saldo: $Y"
- [ ] Aviso 30 min antes de que expire un codigo de canje activo

---

### CB-CL-07: Ver catalogo de beneficios (motivacional)
**Como** cliente,
**quiero** ver todos los beneficios disponibles con su costo en pesos,
**para** saber que puedo obtener y motivarme a acumular cashback.

**Criterios de aceptacion:**
- [ ] El cliente selecciona "Ver beneficios" (`cb_list_rewards`) del menu principal
- [ ] Se listan TODAS los beneficios activos del negocio, sin filtrar por saldo
- [ ] Se muestra el saldo actual al inicio: "Tu saldo: $X MXN."
- [ ] Cada beneficio muestra: nombre y costo en pesos
- [ ] Si el beneficio es alcanzable: "disponible"
- [ ] Si NO es alcanzable: "te faltan $X" — motiva al usuario
- [ ] Al final: "Sigue acumulando cashback para desbloquear mas beneficios."
- [ ] Ejecucion directa — respuesta como texto, NO lista interactiva

**Diferencia con CB-CL-03:** CB-CL-03 filtra por saldo y presenta lista interactiva seleccionable. CB-CL-07 muestra catalogo completo como texto informativo.

---

### CB-CL-08: Dejar feedback
Compartido con earn-burn (CL-10). El mismo comando `submit_feedback` sirve para ambos modulos ya que el feedback se asocia al `customer_id`, no al tipo de programa.

---

## Colaborador

### CB-CO-01: Acreditar cashback a un cliente
**Como** colaborador,
**quiero** acreditar cashback a un cliente basado en su compra,
**para** que reciba el porcentaje correspondiente de su gasto.

**Criterios de aceptacion:**
- [ ] El colaborador indica el OTP temporal del cliente (6 chars)
- [ ] Se valida el OTP en Redis → resuelve a client_id
- [ ] El flujo solicita foto del ticket
- [ ] AI procesa la foto para extraer el monto (unico uso de AI)
- [ ] Si la foto no es legible, max 3 intentos, luego ingreso manual
- [ ] El cashback se calcula: `floor(monto * cashback_rate * 100) / 100`
- [ ] Ejemplo: $2,000 MXN * 5% = $100.00 MXN de cashback
- [ ] Se registra la transaccion con `correctable_until = NOW() + 2h`
- [ ] `purchase_amount` almacena el monto original de la factura (auditoria)
- [ ] Flag `manual_entry` si el monto se ingreso manualmente
- [ ] Se actualiza el saldo atomicamente (TX de Postgres)
- [ ] La respuesta incluye: cashback acreditado, nuevo saldo
- [ ] Se notifica al cliente

**Ref:** Diagrama 2 (Acreditar Cashback)

---

### CB-CO-02: Consultar saldo de un cliente
**Como** colaborador,
**quiero** consultar el saldo y movimientos de cashback de un cliente,
**para** verificar su estado.

**Criterios de aceptacion:**
- [ ] El colaborador proporciona el OTP del cliente
- [ ] Se valida el OTP → resuelve a client_id
- [ ] La respuesta incluye: nombre del cliente, saldo total en pesos, ultimos 10 movimientos
- [ ] Cada movimiento muestra: tipo (earn/burn/adjustment), monto en pesos, fecha

**Ref:** Diagrama 7 (Consultar Saldo Cliente)

---

### CB-CO-03: Corregir cashback (ventana de 2 horas)
**Como** colaborador,
**quiero** corregir una transaccion de cashback reciente,
**para** rectificar errores dentro de las 2 horas permitidas.

**Criterios de aceptacion:**
- [ ] El colaborador identifica al cliente con OTP vigente
- [ ] Solo se muestran transacciones con `correctable_until > NOW()`
- [ ] El colaborador indica el **monto correcto de la factura** (no el cashback directamente)
- [ ] El sistema **recalcula el cashback** con el nuevo monto: `new_cashback = floor(new_amount * rate * 100) / 100`
- [ ] Se crea transaccion de tipo `adjustment` con la diferencia
- [ ] **RN-01**: Si el ajuste resultaria en saldo < 0, se rechaza
- [ ] Se solicita evidencia del error y comentario
- [ ] Se notifica al cliente del ajuste

**Nota:** A diferencia de earn-burn donde se corrige en puntos directamente, en cashback se corrige el **monto de la factura** y el sistema recalcula el cashback automaticamente.

**Ref:** Diagrama 6 (Correccion)

---

### CB-CO-04: Confirmar canje de beneficio
**Como** colaborador,
**quiero** validar el codigo de canje que me presenta el cliente,
**para** confirmar la entrega del beneficio.

**Criterios de aceptacion:**
- [ ] El colaborador proporciona el codigo de canje
- [ ] Se valida en Redis: `GETDEL otp:{code}`, verifica `type == "cb_redemption"`. Fallback a Postgres
- [ ] Si valido: marcar como `confirmed`, registrar `confirmed_by` y `confirmed_at`
- [ ] La respuesta incluye: nombre del beneficio
- [ ] Si invalido/expirado/ya usado: error claro
- [ ] Se notifica al cliente

**Ref:** Diagrama 3 Fase 2

---

### CB-CO-05: Procesar carga de cashback con codigo del cliente
**Como** colaborador,
**quiero** procesar una carga de cashback usando el codigo del cliente,
**para** acreditarle el cashback correspondiente a su compra.

**Criterios de aceptacion:**
- [ ] Se valida el codigo en Redis: `GETDEL otp:{code}`, verifica `type == "cb_load_points"`
- [ ] El flujo solicita foto del ticket
- [ ] AI procesa la foto para extraer el monto
- [ ] El cashback se calcula con `floor(monto * cashback_rate * 100) / 100`
- [ ] Flag `manual_entry` si es ingreso manual
- [ ] Se registra transaccion y actualiza saldo atomicamente
- [ ] Se notifica al cliente

**Ref:** Diagrama 4 Fase 2

---

## Admin (Negocio B2B)

### CB-AD-01: Gestionar catalogo de beneficios
**Como** admin del negocio,
**quiero** crear, ver, editar y eliminar beneficios de mi catalogo de cashback,
**para** definir los premios que mis clientes pueden obtener con su cashback.

**Criterios de aceptacion:**
- [ ] `POST /api/v1/cashback-programs/:program_id/rewards` — crea beneficio con: nombre, descripcion, costo en pesos
- [ ] `GET /api/v1/cashback-programs/:program_id/rewards` — lista todos
- [ ] `PUT /api/v1/cashback-programs/:program_id/rewards/:id` — edita
- [ ] `DELETE /api/v1/cashback-programs/:program_id/rewards/:id` — soft delete
- [ ] Requiere Bearer token
- [ ] Los costos son en pesos (DECIMAL), no en puntos

---

### CB-AD-02: Consultar saldo y transacciones de un cliente
**Como** admin del negocio,
**quiero** consultar el saldo y movimientos de cashback de cualquier cliente via API,
**para** verificar estados de cuenta.

**Criterios de aceptacion:**
- [ ] `GET /api/v1/cashback-programs/:program_id/clients/:id/balance` — saldo en pesos
- [ ] `GET /api/v1/cashback-programs/:program_id/clients/:id/transactions` — historial
- [ ] Las transacciones incluyen: tipo, monto, purchase_amount, balance_after, fecha, colaborador
- [ ] Requiere Bearer token

---

### CB-AD-03: Configurar programa de cashback
**Como** admin del negocio,
**quiero** crear y configurar un programa de cashback con su porcentaje,
**para** definir cuanto cashback reciben mis clientes.

**Criterios de aceptacion:**
- [ ] `POST /api/v1/cashback-programs` — crea programa con: nombre, cashback_rate (ej: 0.05 = 5%)
- [ ] `GET /api/v1/cashback-programs` — lista programas
- [ ] `PUT /api/v1/cashback-programs/:id` — edita (nombre, rate, activo)
- [ ] El cashback_rate se valida: debe ser > 0 y <= 1 (0% a 100%)
- [ ] Requiere Bearer token

---

## Matriz de Trazabilidad

| Historia | Diagrama | Operacion | Menu / Comando | API REST |
|----------|----------|-----------|----------------|----------|
| CB-CL-01 | — | registro (compartido) | — | — |
| CB-CL-02 | 5 | consultar saldo | `cb_check_balance` (ejecucion directa) | GET /clients/:id/balance |
| CB-CL-03 | 3 | canjear - listar filtrado | `cb_redeem` (lista interactiva filtrada) | — |
| CB-CL-04 | 3 | canjear - codigo | `cb_request_redemption` (1 paso: confirmar; reward_id pre-cargado) | — |
| CB-CL-05 | 4 | cargar cashback | `cb_load_request` (ejecucion directa) | — |
| CB-CL-06 | 2, 3, 6 | notificaciones | — | — |
| CB-CL-07 | — | catalogo beneficios | `cb_list_rewards` (ejecucion directa, texto) | GET /rewards |
| CB-CL-08 | — | feedback (compartido) | `submit_feedback` | GET /feedback |
| CB-CO-01 | 2 | acreditar cashback | `cb_add_cashback` (flujo: OTP → foto/monto) | — |
| CB-CO-02 | 7 | consultar saldo cliente | `cb_list_balance` (flujo: OTP → mostrar) | GET /clients/:id/transactions |
| CB-CO-03 | 6 | corregir cashback | `cb_update_cashback` (flujo: OTP → seleccionar → monto factura → evidencia) | — |
| CB-CO-04 | 3 | confirmar canje | `cb_confirm_redemption` (flujo: codigo → confirmar) | — |
| CB-CO-05 | 4 | procesar carga | `cb_load_process` (flujo: codigo → foto/monto) | — |
| CB-AD-01 | — | crud beneficios | — | CRUD /cashback-programs/:id/rewards |
| CB-AD-02 | — | consultar cliente | — | GET /cashback-programs/:id/clients/:id/* |
| CB-AD-03 | — | configurar programa | — | CRUD /cashback-programs |
