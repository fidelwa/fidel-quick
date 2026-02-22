# Cashback

Este sistema el cliente final acumula **saldo en dinero (pesos)** como recompensa por cada compra en el establecimiento (nuestro cliente B2B). En lugar de puntos, recibe un porcentaje del monto de su compra como credito cashback.

El porcentaje de cashback depende de la configuracion del programa (ej: 5% de cashback). Por ejemplo, si el cliente gasta $1,000 MXN con un 5% de cashback, recibe $50 MXN de saldo cashback.

> En algun momento podriamos llegar a tener diferentes porcentajes por categoria de producto o por horario, pero por el momento es un porcentaje global por programa.

Para que este sistema pueda funcionar necesita la misma infraestructura base que earn-burn:
* **1 solo numero de WhatsApp de plataforma** compartido por todos los negocios
    * tanto colaboradores como clientes escriben al mismo numero
    * el sistema identifica automaticamente el rol (cliente o colaborador) y el negocio en contexto
    * cada negocio tiene un **QR con landing page** (`/unirse/:slug`) — la misma landing page sirve para todos los programas
    * si un usuario esta en multiples negocios, se le presenta un menu de seleccion
    * las sesiones se mantienen en Redis (TTL 30min)

**Diferencia clave con earn-burn:** la unidad de balance es **pesos (dinero)**, no puntos. El cliente ve su saldo en pesos y las recompensas se miden en pesos.

## Operaciones:
Las operaciones desde WhatsApp son las mismas que earn-burn pero con terminologia de cashback:
1. consultar-saldo (balance en pesos)
2. cargar-cashback (acreditar cashback por compra)
3. corregir-cashback (ajuste dentro de ventana)
4. canjear-cashback (usar saldo para obtener beneficios)

### Negocio:
Las operaciones del negocio son desde la interfaz REST API:
* crud-colaborador
* crud-programa de cashback (con `cashback_rate` en porcentaje)
* crud-catalogo de beneficios (recompensas medidas en pesos)

### Colaborador:
* acreditar-cashback: le acredita el cashback a un usuario:
    * codigo-usuario (OTP de identidad)
    * foto-factura
    * se calcula el cashback basado en el monto de la factura y el `cashback_rate` del programa
    * ej: factura $2,000 MXN * 5% = $100 MXN de cashback
* consultar-saldo-cliente:
    * codigo usuario
* corregir-cashback:
    * en caso de una correccion
    * con codigo usuario
    * ventana de 2 horas para correcciones
* confirmar-canje:
    * codigo de canje del usuario
    * este valor se carga automaticamente

### Usuario:
* consultar-saldo:
    * ver cuanto cashback tiene acumulado (en pesos)
    * ver ultimos movimientos
* ver-beneficios (catalogo motivacional):
    * lista TODOS los beneficios disponibles con su costo en pesos
    * muestra cuales puede pagar y cuales le faltan
* canjear-beneficio (lista interactiva filtrada):
    * filtra solo los beneficios que el usuario puede pagar con su saldo
    * los presenta como lista interactiva de WhatsApp
    * al seleccionar, pide confirmacion
    * genera codigo de canje valido por 1 hora
* solicitar-carga:
    * genera codigo temporal para que el colaborador acredite el cashback
* dejar-feedback:
    * comentario sobre el establecimiento
