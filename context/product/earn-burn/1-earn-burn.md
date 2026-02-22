# Earn and Burn:

Este sistema el cliente final acumula puntos en el sistema de fidelidad del establecimiento (nuestro cliente B2B), por cada compra canjea una recompensa.

Este sistema puede tener 1 punto por cada 10 pesos mexicano, esto va a depender de la configuracion que se ponga pero por el momento es una proporcion de puntos.

> en algun momento podriamos a llegar a tener diferentes cantidad de puntos por producto, esto puede avanzar y crecer mucho, por el momento no.

Para que este sistema pueda funcionar necesitamos:
* **1 solo numero de WhatsApp de plataforma** compartido por todos los negocios
    * tanto colaboradores como clientes escriben al mismo numero
    * el sistema identifica automaticamente el rol (cliente o colaborador) y el negocio en contexto
    * cada negocio tiene un **QR con landing page** (`/unirse/:slug`) que redirige a WhatsApp con un mensaje natural pre-escrito (ej: "Quiero unirme a Café Roma")
    * si un usuario esta en multiples negocios, se le presenta un menu de seleccion
    * las sesiones se mantienen en Redis (TTL 30min) para recordar el contexto del negocio activo

Todos los datos durante el proceso se deben guardar.

> Analizar cuales webhooks se pueden usar en cada uno de los sistemas en base a las operaciones para poder tener metricas en tiempo real.

> Estaba pensando en ciertas configuraciones que aveces tiene que hacer el admin (dueno de negocio) para controlar sus sistema, y no tiene una pc cerca, deberia poder hacer como un inicio de sesion desde whatsapp, y poder hacer ajustes desde whatsapp rapidamente, como la creacion de un empleado nuevo.

## Operaciones:
las operaciones en general que se pueden hacer desde whatsapp:
1. listar-puntos
2. cargar-puntos
3. actualizar-puntos
4. redimir-puntos
    
### Negocio:
Las operaciones de negico son desde la interfaz del negocio, no desde negocio, las operaciones que se van a poder hacer desde la plaraforma:
* crud-operador
* crud-sistemas de fidel;izacion

### Colaborador:
* agrega-puntos: le agrega los puntos un usuariuo:
    * codigo-usuario
    * foto-factura
    * se agregan de dependiendo la factura, y la proporcion de puntos configurada.
* lista-puntos: 
    * codigo usuario
* actualizar-puntos: 
    * en caso de una correccion
    * con codigo usuario:
    * revision manual del colaborador.
    * se tiene una brecha de tiempo para poder hacer updates de puntos 2 horas
        * se hace en el establecimiento
    * se debe aclara como hacer actualizacion de puntos online
        * como una pantalla, donde muestra la factura.
        * y es como un proceso de chat o de dialogo
* remision-puntos:
    * codigo remision-usuario
    * este valor se debe cargar a la plataforma automaticamente

### Usuario:
cuenta business para interaccion directa para tus usuarios, donde podran interactuar con el sistema de fidelacion y poder llevar registro des sus interacciones dependiendo del tipo de sistema de fidelizacion.

> Para este punto el usuario, en algun momento debio haber escaneado el qr del establecimiento para poder entrar al sistema de fidelidad.

* check-points:
    * que es la forma en la que el usuario puede confirmar los puntos que tiene con tu negocio
* ver-recompensas (catalogo motivacional):
    * lista TODAS las recompensas activas del negocio como texto informativo
    * muestra el balance del usuario al inicio
    * por cada recompensa muestra: nombre, costo en puntos, y status ("disponible" o "te faltan X pts")
    * motiva al usuario a seguir acumulando puntos
* canjear-recompensa (lista interactiva filtrada):
    * filtra solo las recompensas que el usuario puede pagar con su balance actual
    * las presenta como **lista interactiva de WhatsApp** (cada una es seleccionable)
    * si no tiene puntos suficientes para ninguna, le indica su balance y lo motiva a seguir acumulando
    * al seleccionar una recompensa de la lista interactiva:
        * el sistema detecta la seleccion (`reward:{id}`) y pre-carga el reward_id
        * el bot solicita confirmacion: "Confirmas el canje? (Si/No)"
        * usuario confirma
        * se genera un codigo de redencion valido por 1 hora
        * usuario habla con colaborador para pedirle redimir su recompensa
        * colaborador, inicia proceso de confirmar canje
        * colaborador, pide codigo de redencion
        * se hace el proceso, y el sistema internamente genera la reclamacion de la recompensa
        * el sistema genera un id, y un mensaje de confirmacion

> este codigo debe ser anti-maquinas cuanticas.

* el primer ingreso del usuario, le dice hola como estas... este es tu codigo nuevo
* carga-puntos:
    * el usuario/colaborador sugiere hacer la retencion de puntos
    * el usuario crea el codigo con la instruccion 'cargar-puntos' y este le genera un codigo que dura 1h
    * el usuario le pasa le pasa el codigo al colaborador
    * el colaborador inicia el proceso de 'carga-puntos' desde su whatsapp
        * se le pide tomar la foto del ticket de la compra del usuario
        * luego el codigo del usuario
        * confirma cargar x puntos basado en total y de la factura.