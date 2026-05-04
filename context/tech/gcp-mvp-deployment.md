# Despliegue MVP en GCP - Costos Minimos

## Objetivo

Desplegar fidel-quick (Go API + Admin React + PostgreSQL + Redis) en GCP con el menor costo posible para validar el producto con primeros clientes (1-10 negocios, <1000 usuarios finales).

---

## Arquitectura Propuesta

```
                    Internet
                       |
               [Cloud Run] -----> fidel-quick (Go API + Admin SPA)
                   |       \
                   |        \---> WhatsApp Webhook (Meta)
                   |
          +--------+--------+
          |                  |
   [Cloud SQL - Postgres]  [Memorystore Redis]
          |
    [Cloud Storage]  (fotos de tickets)
```

---

## Componentes y Costos Estimados

### 1. Cloud Run (API + Admin) — ~$0/mes

El binario Go sirve tanto la API REST como el admin dashboard (archivos estaticos embebidos o servidos por Gin).

| Recurso | Configuracion | Costo |
|---------|---------------|-------|
| Cloud Run | 1 instancia, 256MB RAM, 1 vCPU | **Free tier: 2M requests/mes** |
| Concurrencia | max-instances=1, min-instances=0 | Scale to zero cuando no hay trafico |

**Config recomendada:**
```
--memory=256Mi
--cpu=1
--max-instances=2
--min-instances=0
--concurrency=80
--timeout=60s
--port=8080
```

**Costo MVP:** $0 (free tier cubre ~2M requests/mes, mas que suficiente para MVP).

> **Nota:** `min-instances=0` significa cold starts de ~1-2s. Aceptable para MVP. Si el webhook de WhatsApp necesita respuesta rapida, subir a `min-instances=1` (~$5/mes).

---

### 2. Cloud SQL PostgreSQL — ~$7-9/mes

| Recurso | Configuracion | Costo |
|---------|---------------|-------|
| Instancia | `db-f1-micro` (0.6GB RAM, shared vCPU) | ~$7.67/mes |
| Storage | 10GB SSD (minimo) | ~$1.70/mes |
| Backups | Automaticos, 7 dias retencion | Incluido |

**Alternativa aun mas barata — AlloyDB Omni o Neon:**
- **Neon (serverless Postgres):** Free tier con 0.5GB storage, 190 horas compute/mes. Costo: **$0/mes** para MVP.
- Si prefieres mantenerlo 100% en GCP, `db-f1-micro` es la opcion mas barata.

**Config recomendada:**
```
gcloud sql instances create fidel-db \
  --database-version=POSTGRES_16 \
  --tier=db-f1-micro \
  --region=us-central1 \
  --storage-size=10GB \
  --storage-type=SSD \
  --backup-start-time=03:00 \
  --availability-type=zonal
```

**Costo MVP:** ~$9/mes (db-f1-micro + 10GB SSD).

---

### 3. Redis — ~$0/mes (alternativa) o ~$11/mes

**Opcion A: Memorystore Redis (managed) — ~$11/mes**

| Recurso | Configuracion | Costo |
|---------|---------------|-------|
| Instancia | Basic tier, 1GB M1 | ~$11/mes |

**Opcion B (recomendada MVP): Redis en Cloud Run sidecar — $0**

Usar Upstash Redis (serverless) con free tier:
- 10,000 commands/dia
- 256MB storage
- Costo: **$0/mes**

Upstash es compatible con el cliente `go-redis` que ya usas. Solo cambia `REDIS_URL`.

**Opcion C: Redis en la misma instancia Cloud Run**

No recomendado por la naturaleza stateless de Cloud Run.

**Costo MVP recomendado:** $0 (Upstash free tier).

---

### 4. Cloud Storage (fotos de tickets) — ~$0/mes

| Recurso | Configuracion | Costo |
|---------|---------------|-------|
| Bucket | Standard, us-central1 | Free tier: 5GB |
| Operaciones | Class A: 5,000/mes, Class B: 50,000/mes | Free tier |

Reemplaza MinIO por Cloud Storage. El SDK de Go (`cloud.google.com/go/storage`) o el compatible con S3 (`aws-sdk-go` con endpoint de GCS) funciona directamente.

**Config:**
```
gsutil mb -l us-central1 gs://fidel-loyalty-invoices
```

**Costo MVP:** $0 (free tier cubre el uso de MVP).

---

### 5. Secret Manager — ~$0/mes

Almacena todos los secrets (tokens WhatsApp, JWT, API keys) de forma segura.

| Recurso | Costo |
|---------|-------|
| 6 secrets | Free tier: 6 versiones activas |
| 10,000 accesos/mes | Free tier |

**Secrets a almacenar:**
```
WHATSAPP_API_TOKEN
WHATSAPP_VERIFY_TOKEN
JWT_SECRET
BEARER_TOKEN
ANTHROPIC_API_KEY
DATABASE_URL (con password)
```

**Costo MVP:** $0.

---

### 6. Artifact Registry — ~$0/mes

Almacena las imagenes Docker del build.

| Recurso | Costo |
|---------|-------|
| 500MB storage | Free tier: 0.5GB |

**Costo MVP:** $0.

---

### 7. Dominio + SSL

| Recurso | Costo |
|---------|-------|
| Cloud Run URL | `*.run.app` gratis con SSL |
| Dominio custom | ~$12/ano (opcional) |
| SSL custom | Automatico con Cloud Run |

Para MVP usar la URL de Cloud Run directamente. WhatsApp webhook acepta cualquier HTTPS.

---

## Resumen de Costos Mensuales

| Servicio | Opcion economica | Costo/mes |
|----------|-----------------|-----------|
| Cloud Run (API) | Free tier | $0 |
| Cloud SQL Postgres | db-f1-micro | $9 |
| Redis | Upstash free tier | $0 |
| Cloud Storage | Free tier | $0 |
| Secret Manager | Free tier | $0 |
| Artifact Registry | Free tier | $0 |
| **Total** | | **~$9/mes** |

> **Alternativa minima absoluta:** Usando Neon (Postgres serverless free tier) + Upstash (Redis free tier) = **$0/mes**. Pero ambos servicios son externos a GCP.

---

## Paso a Paso: Despliegue

### Prerrequisitos

```bash
# Instalar gcloud CLI
brew install google-cloud-sdk

# Autenticarse
gcloud auth login
gcloud config set project fidel-mvp

# Habilitar APIs necesarias
gcloud services enable \
  run.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  redis.googleapis.com \
  vpcaccess.googleapis.com
```

### Paso 1: Crear Cloud SQL

```bash
# Crear instancia
gcloud sql instances create fidel-db \
  --database-version=POSTGRES_16 \
  --tier=db-f1-micro \
  --region=us-central1 \
  --storage-size=10GB \
  --storage-type=SSD \
  --availability-type=zonal

# Crear base de datos
gcloud sql databases create loyalty --instance=fidel-db

# Crear usuario
gcloud sql users create loyalty \
  --instance=fidel-db \
  --password=<GENERA_PASSWORD_SEGURO>
```

### Paso 2: Crear bucket para fotos

```bash
gsutil mb -l us-central1 gs://fidel-loyalty-invoices
```

### Paso 3: Configurar secrets

```bash
# Crear cada secret
echo -n "tu_token_api_whatsapp" | \
  gcloud secrets create WHATSAPP_API_TOKEN --data-file=-

echo -n "tu_verify_token" | \
  gcloud secrets create WHATSAPP_VERIFY_TOKEN --data-file=-

echo -n "tu_jwt_secret_seguro" | \
  gcloud secrets create JWT_SECRET --data-file=-

echo -n "tu_bearer_token" | \
  gcloud secrets create BEARER_TOKEN --data-file=-

echo -n "sk-ant-xxxxx" | \
  gcloud secrets create ANTHROPIC_API_KEY --data-file=-
```

### Paso 4: Crear Artifact Registry

```bash
gcloud artifacts repositories create fidel-repo \
  --repository-format=docker \
  --location=us-central1
```

### Paso 5: Build y push de la imagen

```bash
# Build con Cloud Build (no necesitas Docker local)
gcloud builds submit --tag \
  us-central1-docker.pkg.dev/fidel-mvp/fidel-repo/fidel-quick:latest
```

### Paso 6: Ejecutar migraciones

```bash
# Conectar via Cloud SQL Proxy
cloud-sql-proxy fidel-mvp:us-central1:fidel-db &

# Ejecutar migraciones
migrate -path migrations \
  -database "postgres://loyalty:<PASSWORD>@localhost:5432/loyalty?sslmode=disable" up
```

### Paso 7: Deploy a Cloud Run

**Plan A (MVP minimo — Upstash Redis):**
```bash
gcloud run deploy fidel-quick \
  --image=us-central1-docker.pkg.dev/fidel-mvp/fidel-repo/fidel-quick:latest \
  --region=us-central1 \
  --memory=256Mi \
  --cpu=1 \
  --max-instances=2 \
  --min-instances=0 \
  --concurrency=80 \
  --timeout=60s \
  --port=8080 \
  --allow-unauthenticated \
  --add-cloudsql-instances=fidel-mvp:us-central1:fidel-db \
  --set-env-vars="ENV=production,PORT=8080,S3_BUCKET=fidel-loyalty-invoices,S3_REGION=us-central1,REDIS_URL=<UPSTASH_REDIS_URL>" \
  --set-secrets="WHATSAPP_API_TOKEN=WHATSAPP_API_TOKEN:latest,WHATSAPP_VERIFY_TOKEN=WHATSAPP_VERIFY_TOKEN:latest,JWT_SECRET=JWT_SECRET:latest,BEARER_TOKEN=BEARER_TOKEN:latest,ANTHROPIC_API_KEY=ANTHROPIC_API_KEY:latest" \
  --set-env-vars="DATABASE_URL=postgres://loyalty:<PASSWORD>@/loyalty?host=/cloudsql/fidel-mvp:us-central1:fidel-db"
```

**Plan B/C (GCP Standard — Memorystore Redis):**

Requiere primero crear el VPC connector y la instancia Redis:
```bash
# 1. Crear VPC connector para que Cloud Run acceda a Memorystore
gcloud compute networks vpc-access connectors create fidel-connector \
  --region=us-central1 \
  --range=10.8.0.0/28

# 2. Crear Memorystore Redis
gcloud redis instances create fidel-redis \
  --size=1 \
  --region=us-central1 \
  --tier=basic \
  --redis-version=redis_7_2

# 3. Obtener la IP de Redis
gcloud redis instances describe fidel-redis --region=us-central1 --format="value(host)"
# Ejemplo resultado: 10.0.0.3

# 4. Deploy con VPC connector y Memorystore
gcloud run deploy fidel-quick \
  --image=us-central1-docker.pkg.dev/fidel-mvp/fidel-repo/fidel-quick:latest \
  --region=us-central1 \
  --memory=512Mi \
  --cpu=1 \
  --max-instances=3 \
  --min-instances=1 \
  --concurrency=80 \
  --timeout=60s \
  --port=8080 \
  --cpu-boost \
  --allow-unauthenticated \
  --vpc-connector=fidel-connector \
  --add-cloudsql-instances=fidel-mvp:us-central1:fidel-db \
  --set-env-vars="ENV=production,PORT=8080,S3_BUCKET=fidel-loyalty-invoices,S3_REGION=us-central1,REDIS_URL=redis://10.0.0.3:6379" \
  --set-secrets="WHATSAPP_API_TOKEN=WHATSAPP_API_TOKEN:latest,WHATSAPP_VERIFY_TOKEN=WHATSAPP_VERIFY_TOKEN:latest,JWT_SECRET=JWT_SECRET:latest,BEARER_TOKEN=BEARER_TOKEN:latest,ANTHROPIC_API_KEY=ANTHROPIC_API_KEY:latest" \
  --set-env-vars="DATABASE_URL=postgres://loyalty:<PASSWORD>@/loyalty?host=/cloudsql/fidel-mvp:us-central1:fidel-db"
```

### Paso 8: Configurar WhatsApp Webhook

1. Ir a Meta for Developers > tu app > WhatsApp > Configuration
2. Callback URL: `https://fidel-quick-xxxxx-uc.a.run.app/webhook`
3. Verify token: el mismo que guardaste en secrets
4. Suscribir a: `messages`

---

## Pipeline CI/CD (opcional, recomendado)

Crear `.github/workflows/deploy.yml` o usar Cloud Build triggers:

```yaml
# .github/workflows/deploy.yml
name: Deploy to Cloud Run

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write

    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ secrets.WIF_PROVIDER }}
          service_account: ${{ secrets.SA_EMAIL }}

      - uses: google-github-actions/setup-gcloud@v2

      - name: Build and push
        run: |
          gcloud builds submit --tag \
            us-central1-docker.pkg.dev/${{ secrets.GCP_PROJECT }}/fidel-repo/fidel-quick:${{ github.sha }}

      - name: Deploy
        run: |
          gcloud run deploy fidel-quick \
            --image=us-central1-docker.pkg.dev/${{ secrets.GCP_PROJECT }}/fidel-repo/fidel-quick:${{ github.sha }} \
            --region=us-central1
```

---

## Arquitectura del Despliegue: Que Vive Donde

Todo vive en **un solo contenedor Cloud Run** que sirve por el puerto 8080:

```
Cloud Run — fidel-quick (:8080)
│
├── /webhook              [BOT]       WhatsApp webhook (publico, Meta lo llama)
│   ├── GET  /webhook     Verificacion del webhook (challenge de Meta)
│   └── POST /webhook     Recibe mensajes → flow engine → respuesta WhatsApp
│
├── /unirse/:slug         [LANDING]   Pagina de deeplink para unirse a negocio
│
├── /admin/*              [UI]        Admin dashboard React (archivos estaticos)
│   └── SPA servida por embed.FS de Go
│
├── /api/v1/auth/*        [AUTH]      Login y registro de admins (publico)
│   ├── POST /login
│   └── POST /register
│
├── /api/v1/*             [REST API]  Endpoints protegidos (JWT/Bearer)
│   ├── /programs/*                   Earn-burn CRUD
│   ├── /cashback-programs/*          Cashback CRUD
│   ├── /customers/*                  Negocios, colaboradores, clientes
│   └── ...
│
└── /api/docs             [DOCS]      Swagger UI + OpenAPI spec
```

### Por que todo en un solo contenedor

1. **Un solo deploy** — `docker build` + `gcloud run deploy` y listo
2. **Cero costo extra** — no hay hosting separado para la UI
3. **Sin CORS** — el admin y la API estan en el mismo origen
4. **Simplicidad** — un repo, un Dockerfile, un pipeline

### Como funciona

El admin dashboard (React/Vite) se compila a archivos estaticos (~2MB) y se embebe dentro del binario Go usando `embed.FS`. El proceso Go sirve todo:

```
[Vite build] → admin/dist/ → [embed.FS en Go] → Cloud Run sirve /admin/*
                                                → Cloud Run sirve /api/*
                                                → Cloud Run sirve /webhook
```

**Flujo de build:**

```dockerfile
# Stage 1: Build admin React
FROM node:20-alpine AS admin-builder
WORKDIR /admin
COPY admin/package*.json ./
RUN npm ci
COPY admin/ .
RUN npm run build     # Genera admin/dist/

# Stage 2: Build Go binary (con admin embebido)
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=admin-builder /admin/dist ./admin/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /fidel-quick .

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /fidel-quick /fidel-quick
EXPOSE 8080
CMD ["/fidel-quick"]
```

**Codigo Go para servir el admin:**

```go
// En api/router.go o main.go
import "embed"

//go:embed admin/dist/*
var adminFS embed.FS

// En SetupRouter():
// Servir SPA — todos los paths bajo /admin sirven index.html (client-side routing)
adminSub, _ := fs.Sub(adminFS, "admin/dist")
r.StaticFS("/admin", http.FS(adminSub))
r.NoRoute(func(c *gin.Context) {
    if strings.HasPrefix(c.Request.URL.Path, "/admin") {
        c.FileFromFS("/", http.FS(adminSub))
        return
    }
    c.JSON(404, gin.H{"error": "not found"})
})
```

### Cuando separar (futuro)

Separar el admin a Firebase Hosting o Cloud Storage + CDN solo cuando:
- El bundle supere ~10MB y quieras CDN global
- Necesites deploys independientes del backend
- Tengas equipos separados para frontend y backend

Para MVP no hay razon para separar.

---

## Consideraciones de Seguridad

1. **Cloud SQL:** Solo accesible via Cloud SQL Proxy (no IP publica)
2. **Secrets:** Nunca en variables de entorno planas, siempre Secret Manager
3. **Cloud Run:** `--allow-unauthenticated` solo porque el webhook de WhatsApp necesita acceso publico. La API REST esta protegida por Bearer/JWT a nivel de aplicacion
4. **HTTPS:** Automatico en Cloud Run

---

## Comparativa de Planes: MVP Minimo vs GCP Standard

### Plan A: MVP Minimo (servicios externos + free tiers)

Optimizado para gastar lo menos posible. Usa Upstash (externo) para Redis.

| Servicio | Solucion | Costo/mes |
|----------|----------|-----------|
| Cloud Run | Free tier, min=0, 256MB | $0 |
| Cloud SQL Postgres | db-f1-micro (0.6GB shared) | $9 |
| Redis | Upstash free tier (externo) | $0 |
| Cloud Storage | Free tier 5GB | $0 |
| Secret Manager | Free tier | $0 |
| Artifact Registry | Free tier | $0 |
| Dominio | *.run.app | $0 |
| **Total** | | **~$9/mes** |

**Pros:** Costo minimo, suficiente para validar con 1-5 negocios.
**Contras:** Cold starts de 1-2s, Redis externo agrega latencia (~10-30ms), Postgres shared puede tener picos de lentitud, limite de 10K commands/dia en Upstash.

---

### Plan B: GCP Standard (todo en GCP, sin free tiers externos)

Todo dentro de GCP con recursos dedicados. Sin cold starts, Redis managed, Postgres con recursos propios. Listo para produccion real.

| Servicio | Solucion | Costo/mes |
|----------|----------|-----------|
| Cloud Run | min=1, max=3, 512MB, 1 vCPU | ~$15 |
| Cloud SQL Postgres | db-g1-small (1.7GB, 1 shared vCPU) | ~$26 |
| Cloud SQL Storage | 10GB SSD + backups automaticos | ~$2 |
| Memorystore Redis | Basic tier, 1GB M1 | ~$11 |
| Cloud Storage | Standard, 10GB estimado | ~$0.20 |
| Secret Manager | 6 secrets | $0 |
| Artifact Registry | ~1GB imagenes | ~$0.10 |
| Cloud Armor (WAF) | Opcional, reglas basicas | $0 (free tier) |
| Dominio custom | fidel.app (si ya lo tienen) | ~$1/mes (~$12/ano) |
| SSL certificado | Managed por Cloud Run | $0 |
| Cloud Logging | 50GB/mes free tier | $0 |
| Cloud Monitoring | Basic | $0 |
| **Total** | | **~$55/mes** |

**Config Cloud Run (Plan B):**
```
--memory=512Mi
--cpu=1
--max-instances=3
--min-instances=1
--concurrency=80
--timeout=60s
--port=8080
--cpu-boost
```

**Config Cloud SQL (Plan B):**
```bash
gcloud sql instances create fidel-db \
  --database-version=POSTGRES_16 \
  --tier=db-g1-small \
  --region=us-central1 \
  --storage-size=10GB \
  --storage-type=SSD \
  --backup-start-time=03:00 \
  --maintenance-window-day=SUN \
  --maintenance-window-hour=04 \
  --availability-type=zonal \
  --insights-config-query-insights-enabled \
  --insights-config-record-application-tags
```

**Config Memorystore Redis (Plan B):**
```bash
gcloud redis instances create fidel-redis \
  --size=1 \
  --region=us-central1 \
  --tier=basic \
  --redis-version=redis_7_2
```

> Memorystore crea la instancia en una VPC. Cloud Run necesita un conector VPC para comunicarse:
> ```bash
> gcloud compute networks vpc-access connectors create fidel-connector \
>   --region=us-central1 \
>   --range=10.8.0.0/28
>
> # Agregar al deploy de Cloud Run:
> --vpc-connector=fidel-connector
> ```

**Pros:** Sin cold starts, baja latencia Redis (~1ms vs ~10-30ms), Postgres con mas memoria, todo managed por GCP, metricas y logging integrados, listo para escalar.
**Contras:** ~6x mas caro que Plan A.

---

### Plan C: GCP Intermedio (todo GCP, micro tiers)

Compromiso entre A y B. Todo dentro de GCP pero con los tiers mas economicos.

| Servicio | Solucion | Costo/mes |
|----------|----------|-----------|
| Cloud Run | min=1, max=2, 256MB, 1 vCPU | ~$7 |
| Cloud SQL Postgres | db-f1-micro (0.6GB shared) | ~$9 |
| Memorystore Redis | Basic tier, 1GB M1 | ~$11 |
| Cloud Storage | Standard, free tier | $0 |
| Secret Manager | Free tier | $0 |
| Artifact Registry | Free tier | $0 |
| Dominio | *.run.app | $0 |
| **Total** | | **~$27/mes** |

**Pros:** Todo en GCP, sin cold starts (min=1), Redis managed con baja latencia.
**Contras:** Postgres shared puede ser lento bajo carga.

---

### Tabla Comparativa

| Aspecto | Plan A ($9) | Plan C ($27) | Plan B ($55) |
|---------|-------------|--------------|--------------|
| Cold starts | Si (1-2s) | No | No |
| Latencia Redis | ~10-30ms (externo) | ~1ms (VPC) | ~1ms (VPC) |
| Postgres RAM | 0.6GB shared | 0.6GB shared | 1.7GB dedicado |
| Redis commands | 10K/dia limite | Ilimitado | Ilimitado |
| Dependencias externas | Upstash | Ninguna | Ninguna |
| Logging/Monitoring | Basico | GCP integrado | GCP integrado |
| Negocios soportados | 1-5 | 5-20 | 20-100 |
| Usuarios finales | <500 | <5,000 | <50,000 |
| VPC connector | No necesario | Si (~$0 free tier) | Si (~$0 free tier) |
| Webhook response time | 1-3s (cold) | <200ms | <200ms |

> **Recomendacion:** Iniciar con **Plan A** para validar. Migrar a **Plan C** cuando tengas 3+ negocios activos o cuando los cold starts afecten la experiencia del webhook de WhatsApp. Escalar a **Plan B** cuando superes 10 negocios.

---

## Escalamiento Futuro

Cuando el producto crezca mas alla del MVP:

| Componente | Plan A | Plan C | Plan B | Crecimiento |
|-----------|--------|--------|--------|-------------|
| Cloud Run | min=0, max=2 | min=1, max=2 | min=1, max=3 | min=2, max=10 |
| Cloud SQL | db-f1-micro | db-f1-micro | db-g1-small | db-custom-2-4096 + HA |
| Redis | Upstash free | Memorystore 1GB | Memorystore 1GB | Memorystore 5GB + replica |
| Storage | Free tier | Free tier | Standard | Multi-region |
| Costo | $9/mes | $27/mes | $55/mes | ~$150-300/mes |

La arquitectura no cambia, solo escalan los recursos.

---

## Checklist de Deploy

### Infraestructura base (todos los planes)
- [ ] Crear proyecto GCP y habilitar billing
- [ ] Habilitar APIs (Cloud Run, SQL, Secrets, Artifact Registry, Cloud Build, Redis, VPC Access)
- [ ] Crear instancia Cloud SQL PostgreSQL (db-f1-micro para Plan A/C, db-g1-small para Plan B)
- [ ] Crear bucket Cloud Storage
- [ ] Crear secrets en Secret Manager
- [ ] Crear repositorio en Artifact Registry

### Redis (elegir uno)
- [ ] **Plan A:** Crear cuenta Upstash y obtener REDIS_URL
- [ ] **Plan B/C:** Crear Memorystore Redis + VPC connector

### Aplicacion
- [ ] Build y push de imagen Docker
- [ ] Ejecutar migraciones de base de datos (via Cloud SQL Proxy)
- [ ] Deploy a Cloud Run (con VPC connector si Plan B/C)

### Verificacion
- [ ] Configurar webhook URL en Meta/WhatsApp
- [ ] Verificar webhook con curl de prueba
- [ ] Crear primer admin con `POST /api/v1/auth/register`
- [ ] Probar flujo completo: WhatsApp -> API -> DB -> respuesta
