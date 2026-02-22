# WhatsApp — Como funciona nuestro sistema

## Resumen

Usamos **1 solo numero de WhatsApp** para toda la plataforma. Todos los negocios, colaboradores y clientes interactuan por el mismo numero. El sistema identifica quien es quien y a que negocio pertenece.

## Como llega un cliente nuevo

1. El negocio tiene un **QR** en su establecimiento (mesa, mostrador, ticket, etc.)
2. El cliente escanea el QR con su telefono
3. Se abre una **pagina web intermedia** que muestra:
   - Nombre y logo del negocio
   - Descripcion corta del programa de fidelidad
   - Boton "Unirme por WhatsApp"
4. Al presionar el boton, se abre WhatsApp con un mensaje natural pre-escrito, ejemplo: "Quiero unirme a Café Roma"
5. El cliente presiona enviar
6. El bot lo registra automaticamente en el programa de ese negocio

## Por que una pagina intermedia

- El cliente ve a donde se esta uniendo antes de abrir WhatsApp (confianza)
- Podemos recopilar datos de contexto (que negocio, que campaña, etc.) sin mostrar codigos raros
- Podemos medir cuantas personas escanean vs cuantas realmente se unen
- Si WhatsApp no esta instalado, podemos ofrecer alternativa

## Como identifica el sistema a cada persona

Cuando alguien escribe al numero de la plataforma:

1. Se busca su telefono en la base de datos
2. Se determina su **rol** para el negocio en contexto:
   - **Cliente** → ve sus puntos, rewards, puede canjear
   - **Colaborador** → puede cargar puntos, confirmar canjes
3. Si esta registrado en varios negocios, se le pregunta en cual quiere operar
4. Si no esta registrado, se le pide que escanee el QR del establecimiento
5. Una vez dentro de un negocio, el menu principal muestra **solo las opciones de los programas activos** de ese negocio (ej: si solo tiene cashback, no muestra opciones de puntos)
6. Al final del menu siempre aparece **"Usar otro establecimiento"** para cambiar de negocio sin esperar a que expire la sesion

## Roles y que puede hacer cada uno

### Cliente (usuario final)
- Consultar su balance de puntos
- Ver catalogo de rewards
- Solicitar canje de reward
- Pedir codigo para cargar puntos
- Dejar feedback del establecimiento

### Colaborador (empleado del negocio)
- Cargar puntos a un cliente
- Confirmar canje de reward
- Consultar puntos de un cliente
- Corregir transacciones (ventana de 2 horas)

### Admin (dueno del negocio)
- No usa WhatsApp — administra todo por la API/dashboard web
- Configura rewards, consulta balances, gestiona colaboradores

## Costos de WhatsApp

### Lo que es gratis
- **Todo mensaje que el usuario inicia** (cliente o colaborador escribe primero) es gratis, sin limite
- Esto cubre ~95% de las interacciones del sistema de fidelidad, porque el flujo natural es que el usuario escriba para consultar o operar

### Lo que se paga
- Solo se paga cuando **nosotros enviamos el primer mensaje** (notificaciones push, promos, recordatorios)
- Costo por mensaje, no por conversacion (desde julio 2025)
- Precios varian por pais y tipo de mensaje

### Estrategia de ahorro
- Disenar el sistema para que el usuario siempre inicie la conversacion
- Cuando un cliente escribe, aprovechar la ventana de 24 horas para informarle todo lo pendiente (puntos por vencer, nuevos rewards, etc.) sin costo extra
- Usar QR codes en el establecimiento para que los clientes inicien conversaciones (gratis)
- Reservar los mensajes push pagados solo para marketing real

## Estructura del QR

Cada negocio tiene su propio QR que apunta a:

```
https://fidel.app/unirse/[slug-del-negocio]
```

Ejemplo: `https://fidel.app/unirse/cafe-roma`

Esa pagina muestra la info del negocio y redirige a WhatsApp con un mensaje natural pre-escrito.

## Para colaboradores

El admin registra al colaborador por la API. El colaborador recibe un link/QR interno especifico que lo identifica como empleado del negocio. A partir de ahi, cuando escriba al bot, el sistema sabe que es colaborador y le muestra las herramientas de operacion.

## Numero de WhatsApp dedicado (opcional premium)

Por defecto todos los negocios usan el numero compartido de la plataforma. Si un negocio quiere su propio numero de WhatsApp con su nombre y logo, se ofrece como servicio premium con costo adicional.
