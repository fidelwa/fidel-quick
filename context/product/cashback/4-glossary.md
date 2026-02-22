# Cashback: Glosario

> Terminologia del modulo cashback. Para terminos compartidos (Customer, Cliente, Colaborador, Session, OTP, etc.) ver `../earn-burn/4-glossary.md`.

| Termino | Definicion | Ejemplo |
|---------|-----------|---------|
| **Cashback** | Porcentaje del monto de una compra que se acredita al cliente como saldo en pesos. Es la unidad de valor del modulo (a diferencia de "puntos" en earn-burn) | Compra de $2,000 MXN con 5% = $100 MXN de cashback |
| **Saldo cashback** | Balance acumulado del cliente en pesos MXN. Nunca puede ser menor a $0 (RN-01). Se usa para canjear beneficios | Saldo: $350 MXN |
| **Cashback rate** | Porcentaje de conversion compra → cashback. Configurado por programa. Almacenado como decimal (ej: 0.05 = 5%) | `cashback_rate = 0.05` → 5% de cada compra |
| **Beneficio** | Premio definido por el customer que el cliente puede obtener con su saldo cashback. Equivalente a "recompensa" en earn-burn pero medido en pesos | "Descuento $200" ($200 MXN), "Bebida gratis" ($80 MXN) |
| **Canje cashback** | Proceso en el que el cliente usa saldo cashback para obtener un beneficio. Genera un codigo temporal de 1h. El colaborador lo confirma | "Quiero canjear el Descuento $200" |
| **Codigo de canje cashback** | OTP con `type: "cb_redemption"` (6 chars, 1h TTL, GETDEL). Metadata incluye `{reward_id, amount_spent}` | `M4K7R2` |
| **Codigo de carga cashback** | OTP con `type: "cb_load_points"` (6 chars, 15min TTL, GETDEL). Generado por el cliente para iniciar carga con colaborador | `T3N8P5` |
| **Purchase amount** | Monto original de la compra del cliente (en pesos). Se almacena en la transaccion para auditoria y para recalcular en correcciones | `purchase_amount = 2000.00` |
| **Acreditar cashback** | Operacion del colaborador: procesar ticket de compra → calcular cashback → agregar al saldo del cliente | "Acreditar $100 MXN por compra de $2,000" |
| **Correccion cashback** | Ajuste donde se corrige el **monto de la factura** (no el cashback directamente). El sistema recalcula el cashback con el nuevo monto y aplica la diferencia | Factura de $5,000 → $6,000. Cashback: $250 → $300. Ajuste: +$50 |
| **Programa cashback** | Configuracion de cashback por negocio. Incluye nombre y cashback_rate. Las rutas API se estructuran alrededor de programas | `cashback_programs` table, `/api/v1/cashback-programs/:id/...` |

## Diferencias clave con earn-burn

| Aspecto | Earn-Burn | Cashback |
|---------|-----------|----------|
| **Unidad de balance** | Puntos (enteros) | Pesos MXN (decimal, 2 decimales) |
| **Ratio/Rate** | `points_ratio` (ej: 1000 = 1 pt/$1,000) | `cashback_rate` (ej: 0.05 = 5%) |
| **Calculo** | `floor(monto / points_ratio)` | `floor(monto * rate * 100) / 100` |
| **Recompensas** | Catalogo con costo en puntos | Catalogo con costo en pesos |
| **Correccion** | Se corrige monto en puntos directamente | Se corrige monto de factura; sistema recalcula cashback |
| **Tablas DB** | `points_balances`, `points_transactions`, `rewards`, `redemptions` | `cashback_balances`, `cashback_transactions`, `cashback_rewards`, `cashback_redemptions` |
| **Prefijo comandos** | sin prefijo (ej: `check_points`) | `cb_` (ej: `cb_check_balance`) |
| **Prefijo seleccion** | `reward:` | `benefit:` |
| **OTP types** | `redemption`, `load_points` | `cb_redemption`, `cb_load_points` |
