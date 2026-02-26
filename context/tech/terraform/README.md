# Planes de Infraestructura Terraform — fidel-quick

Tres planes de infraestructura en GCP para fidel-quick, cada uno optimizado para una etapa diferente del producto.

---

## Resumen Rapido

| | Plan A | Plan C | Plan B |
|--|--------|--------|--------|
| **Archivo** | `plan-a-mvp-minimo.tf` | `plan-c-gcp-intermedio.tf` | `plan-b-gcp-standard.tf` |
| **Costo** | ~$9/mes | ~$27/mes | ~$55/mes |
| **Para quien** | Validar idea, 1-5 negocios | Primeros clientes, 5-20 negocios | Produccion real, 20-100 negocios |
| **Redis** | Upstash (externo, gratis) | Memorystore GCP | Memorystore GCP |
| **Postgres** | db-f1-micro (shared) | db-f1-micro (shared) | db-g1-small (dedicado) |
| **Cold starts** | Si (1-2s) | No | No |
| **Monitoring** | No | No | Si (uptime + alertas) |

---

## Estructura de Archivos

```
terraform/
├── README.md                       ← Este archivo
├── terraform.tfvars.example        ← Variables de ejemplo (copiar a .tfvars)
├── plan-a-mvp-minimo.tf            ← Plan A: ~$9/mes
├── plan-c-gcp-intermedio.tf        ← Plan C: ~$27/mes
└── plan-b-gcp-standard.tf          ← Plan B: ~$55/mes
```

---

## Como Usar

### 1. Elegir un plan

Copiar el plan elegido como `main.tf`:

```bash
cd context/tech/terraform/

# Elegir UNO:
cp plan-a-mvp-minimo.tf main.tf        # $9/mes
cp plan-c-gcp-intermedio.tf main.tf     # $27/mes
cp plan-b-gcp-standard.tf main.tf       # $55/mes
```

### 2. Configurar variables

```bash
cp terraform.tfvars.example terraform.tfvars
# Editar terraform.tfvars con tus valores reales
```

### 3. Ejecutar

```bash
terraform init
terraform plan      # Revisar que va a crear
terraform apply     # Crear la infraestructura
```

### 4. Post-deploy (manual)

Terraform crea la infraestructura pero hay pasos manuales despues:

```bash
# 1. Build y push de la imagen Docker
gcloud builds submit --tag $(terraform output -raw container_registry)/fidel-quick:latest

# 2. Conectar a Cloud SQL para ejecutar migraciones
cloud-sql-proxy $(terraform output -raw cloud_sql_connection) &
migrate -path ../../migrations -database "postgres://loyalty:PASSWORD@localhost:5432/loyalty?sslmode=disable" up

# 3. Configurar webhook en Meta for Developers
#    URL: $(terraform output -raw api_url)/webhook
#    Verify token: el que pusiste en WHATSAPP_VERIFY_TOKEN
```

---

## Detalle de Cada Plan

### Plan A: MVP Minimo (~$9/mes)

**Archivo:** `plan-a-mvp-minimo.tf`

**Ideal para:** Validar el producto con los primeros 1-5 negocios. Gastar lo minimo posible mientras se prueba el mercado.

**Recursos que crea:**

| Recurso | Servicio GCP | Tier |
|---------|-------------|------|
| Base de datos | Cloud SQL PostgreSQL 16 | db-f1-micro (0.6GB RAM, shared vCPU) |
| Storage | Cloud Storage bucket | Standard, 10GB |
| Secrets | Secret Manager (6 secrets) | Free tier |
| Docker Registry | Artifact Registry | Free tier |
| API + Bot + Admin | Cloud Run | min=0, max=2, 256MB RAM |
| Service Account | IAM | Cloud SQL + Storage + Secrets |

**NO crea:**
- Memorystore Redis (usa Upstash externo, variable `upstash_redis_url`)
- VPC connector (no necesario sin Memorystore)
- Monitoring/alertas

**Variables especificas:**
```hcl
upstash_redis_url = "rediss://default:xxxxx@us1-xxxxx.upstash.io:6379"
```

**Limitaciones:**
- Cold starts de 1-2s cuando no hay trafico (Cloud Run scale to zero)
- Redis externo agrega ~10-30ms de latencia por request
- Postgres shared puede tener variabilidad de rendimiento
- Upstash free tier: 10,000 commands/dia

**Cuando migrar al Plan C:**
- Cuando los cold starts afecten la experiencia del webhook de WhatsApp
- Cuando superes 10,000 commands/dia de Redis
- Cuando tengas 3+ negocios activos

---

### Plan C: GCP Intermedio (~$27/mes)

**Archivo:** `plan-c-gcp-intermedio.tf`

**Ideal para:** Primeros clientes pagando. Todo dentro de GCP, sin dependencias externas. Sin cold starts.

**Recursos que crea:**

| Recurso | Servicio GCP | Tier |
|---------|-------------|------|
| Base de datos | Cloud SQL PostgreSQL 16 | db-f1-micro (0.6GB RAM, shared vCPU) |
| Redis | Memorystore Redis 7.2 | Basic, 1GB |
| VPC connector | Serverless VPC Access | 10.8.0.0/28 |
| Storage | Cloud Storage bucket | Standard |
| Secrets | Secret Manager (6 secrets) | Free tier |
| Docker Registry | Artifact Registry | Free tier |
| API + Bot + Admin | Cloud Run | min=1, max=2, 256MB RAM |
| Service Account | IAM | Cloud SQL + Storage + Secrets |

**Diferencias vs Plan A:**
- Cloud Run `min=1` — siempre hay una instancia corriendo, sin cold starts
- Memorystore Redis en VPC — latencia ~1ms vs ~10-30ms de Upstash
- VPC connector — permite comunicacion privada Cloud Run <-> Redis
- Sin dependencias externas — todo managed por GCP

**Variables especificas:**
Ninguna adicional. Redis se configura automaticamente con el host de Memorystore.

**Cuando migrar al Plan B:**
- Cuando necesites mas RAM en Postgres (consultas lentas, muchos datos)
- Cuando quieras monitoring y alertas automaticas
- Cuando superes 20 negocios activos

---

### Plan B: GCP Standard (~$55/mes)

**Archivo:** `plan-b-gcp-standard.tf`

**Ideal para:** Produccion real con 20-100 negocios. Recursos dedicados, monitoring, y configuracion lista para escalar.

**Recursos que crea:**

| Recurso | Servicio GCP | Tier |
|---------|-------------|------|
| Base de datos | Cloud SQL PostgreSQL 16 | db-g1-small (1.7GB RAM, 1 vCPU) |
| Redis | Memorystore Redis 7.2 | Basic, 1GB |
| VPC connector | Serverless VPC Access | 10.8.0.0/28 (min=2, max=3 instances) |
| Storage | Cloud Storage bucket | Standard, versioning, lifecycle rules |
| Secrets | Secret Manager (6 secrets) | Free tier |
| Docker Registry | Artifact Registry | Cleanup policy (keep 10 images) |
| API + Bot + Admin | Cloud Run | min=1, max=3, 512MB, cpu-boost |
| Service Account | IAM | SQL + Storage + Secrets + Logging + Monitoring |
| Uptime check | Cloud Monitoring | Health check cada 5 min |
| Alerta latencia | Cloud Monitoring | p95 > 2s |
| Alerta errores | Cloud Monitoring | 5xx > 5% |
| Notificaciones | Cloud Monitoring | Email |

**Diferencias vs Plan C:**

| Aspecto | Plan C | Plan B |
|---------|--------|--------|
| Cloud SQL | db-f1-micro (0.6GB shared) | db-g1-small (1.7GB, query insights, slow query log) |
| Cloud Run | 256MB, max=2 | 512MB, max=3, cpu-boost, liveness probe |
| Storage | 1 lifecycle rule | Versioning + 2 lifecycle rules (Nearline 90d, Coldline 365d) |
| VPC connector | Default instances | min=2, max=3 instances |
| Artifact Registry | Basico | Cleanup policy (keep 10 images) |
| Monitoring | Ninguno | Uptime check + alerta latencia + alerta error rate |
| IAM | Basico | + logging.logWriter + monitoring.metricWriter |
| Terraform backend | Local | GCS recomendado (state compartido) |

**Variables especificas:**
```hcl
# Opcionales en Plan B
custom_domain = "api.fidel.app"     # Dominio custom (opcional)
platform_url  = "https://fidel.app" # URL para deeplinks
alert_email   = "alerts@fidel.app"  # Email para alertas de monitoring
```

**Configuracion de Postgres avanzada:**
- Query Insights habilitado (ver consultas lentas en consola GCP)
- `log_min_duration_statement = 1000` (log queries > 1 segundo)
- Maintenance window: domingos 4 AM
- Point-in-time recovery habilitado (7 dias)
- Disk autoresize habilitado

---

## Migracion Entre Planes

### De Plan A a Plan C

Cambios necesarios:
1. Crear Memorystore Redis + VPC connector (recursos nuevos)
2. Cambiar Cloud Run para usar VPC connector y `REDIS_URL` interno
3. Actualizar `min_instance_count` de 0 a 1
4. Eliminar variable `upstash_redis_url`

```bash
# Opcion 1: Reemplazar el .tf y hacer apply
cp plan-c-gcp-intermedio.tf main.tf
terraform plan    # Revisar cambios
terraform apply

# Opcion 2: Manual — agregar Redis y VPC, luego actualizar Cloud Run
```

> **Nota:** La migracion no tiene downtime. Terraform agrega Redis y VPC connector primero, luego actualiza Cloud Run para usarlos.

### De Plan C a Plan B

Cambios necesarios:
1. Cambiar Cloud SQL de `db-f1-micro` a `db-g1-small` (requiere restart ~2-5 min)
2. Actualizar Cloud Run: 512MB, cpu-boost, liveness probe
3. Agregar monitoring (uptime check + alertas)
4. Agregar cleanup policy a Artifact Registry

```bash
cp plan-b-gcp-standard.tf main.tf
terraform plan
terraform apply
```

> **Nota:** El cambio de tier de Cloud SQL causa un restart de ~2-5 minutos. Planificarlo en ventana de mantenimiento.

---

## Variables Compartidas (Todos los Planes)

| Variable | Descripcion | Sensible |
|----------|-------------|----------|
| `project_id` | ID del proyecto GCP | No |
| `region` | Region GCP (default: us-central1) | No |
| `db_password` | Password de PostgreSQL | Si |
| `whatsapp_api_token` | Token de la API de WhatsApp Business | Si |
| `whatsapp_verify_token` | Token de verificacion del webhook | Si |
| `whatsapp_phone_number_id` | Phone Number ID de WhatsApp | No |
| `whatsapp_display_phone` | Numero de telefono para mostrar | No |
| `jwt_secret` | Secret para firmar JWT | Si |
| `bearer_token` | Token Bearer para API REST | Si |
| `anthropic_api_key` | API key de Anthropic (OCR) | Si |
| `container_image` | URL de la imagen Docker | No |

---

## Outputs (Todos los Planes)

| Output | Descripcion |
|--------|-------------|
| `api_url` | URL del Cloud Run — usar como webhook URL en Meta |
| `cloud_sql_connection` | Connection name para Cloud SQL Proxy |
| `storage_bucket` | Nombre del bucket de Cloud Storage |
| `estimated_monthly_cost` | Costo mensual estimado |
| `redis_host` | Host de Memorystore (solo Plan B/C) |

---

## Seguridad

Todos los planes implementan:

- **Cloud SQL sin IP publica** — solo accesible via Cloud SQL Proxy o Unix socket
- **Secrets en Secret Manager** — nunca en variables de entorno planas
- **Service Account dedicado** — principio de menor privilegio
- **HTTPS automatico** — Cloud Run provee SSL managed
- **Variables sensibles** — marcadas `sensitive = true` en Terraform (no aparecen en logs)

---

## Archivos que NO se commitean

Agregar al `.gitignore`:

```
context/tech/terraform/.terraform/
context/tech/terraform/terraform.tfstate
context/tech/terraform/terraform.tfstate.backup
context/tech/terraform/terraform.tfvars
context/tech/terraform/main.tf
```

Los archivos de estado y variables con secrets nunca deben estar en git.
