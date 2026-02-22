Somos una empresa que vende sistemas de fidelazacion as a service,
todo lo que ofrecemos a nuestros clientes (B2B) son sistemas de fidelizacion.

nuestra principal punto de funcionalidades es en whatsapp, entonces proveemos diferentes servicios al rededor de esto:

Tenemos varias entidades:
1. negocio
2. colaborador
3. cliente

Cada una de las entidades tiene configuraciones y propiedades especiales:

customers:
id
nombre
direccion
telefono
fecha de creacion
fecha de actualizacion
activo

customers_chance_log:
id
id_negocio
campo_actualizado
valor_anterior
valor_nuevo
fecha_actualizacion

customers_configs:
id
id_customer

tiers:
id
name
descripcion
precio
active
fecha_creacion
fecha_update

tiers_change_log
id
id_tier
campo_update
valor_anterior
valor_nuevo
fecha_update

tiers_features_tiers:
id
id_tier
id_feature_tier
activo
fecha_creacion

features_tiers:
id
name
descripcion
fecha_creacion
fecha_actualizacion

## Sistemas de recomendacion:
son todos los sistemas de recomendacion que vamos a tener

tipos:
1. earn-burn
2. cashback
3. tiers
4. checkin
5. gamefitication.

### estructura de tablas:

loyalties:
id
name
description
active
date_create
date_update

loyalties_change_log:
id
id_loyalty
field_name
last_value
new_value
date_update

para cada tipo de loyalty se debe crear una tabla para hacer el seguimiento de sus configuraciones.

## Colaborador:
Son las personas que ayudan y asisten al negocio tienen varias funciones, dentro del establecimiento y ademas hace varias funciones dentro del sistema de fidelidad, dependiendo cual sistema sea.

### tables:

colaborators
id
id_customer
name
hash_id
date_birth
date_creation
date_update
active

colaborators_change_log
id
id_colaborator
field
last_value
new_value
date_update

actions: son las acciones permitidas tomar fotos etc.
id
name
description
active
date_create
date_update

colaborators_actions: acciones permitidas por colaborador:
id
id_colaborator
id_action
active
date_create
date_update

## client:
es el usuario final del establecimiento, es el cliente de nuestro usuario

clients
id
hash
name
phone
date_created
date_updated

clients_change_log
id
id_client
field
last_value
new_value
date_updated