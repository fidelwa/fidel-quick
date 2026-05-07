# Setup del CI/CD — GitHub Actions → Cloud Run

Pasos manuales que se hacen **una sola vez** para que `.github/workflows/deploy.yml` pueda autenticarse contra GCP y desplegar.

## 1. Crear Service Account para CI

```bash
set -a && . ./.env.deploy && set +a

gcloud iam service-accounts create github-deployer \
  --display-name="GitHub Actions deployer" \
  --project="$PROJECT_ID"

CI_SA="github-deployer@${PROJECT_ID}.iam.gserviceaccount.com"
```

## 2. Otorgar permisos al SA

```bash
for role in \
    roles/run.admin \
    roles/cloudbuild.builds.editor \
    roles/artifactregistry.writer \
    roles/iam.serviceAccountUser \
    roles/secretmanager.secretAccessor \
    roles/storage.admin \
    roles/cloudsql.client; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$CI_SA" \
    --role="$role" \
    --quiet
done
```

> `serviceAccountUser` es necesario para que CI pueda actuar como el SA del Cloud Run runtime (`fidel-quick-sa`).

## 3. Generar JSON key y agregarla como secret en GitHub

```bash
gcloud iam service-accounts keys create /tmp/github-deployer.json \
  --iam-account="$CI_SA"

# Copia el contenido completo del JSON al portapapeles:
cat /tmp/github-deployer.json | pbcopy

# IMPORTANTE: borra el archivo del disco después de copiarlo a GitHub
rm /tmp/github-deployer.json
```

En GitHub: **Settings → Secrets and variables → Actions → New repository secret**

- Name: `GCP_SA_KEY`
- Value: pega el JSON completo (incluye llaves `{...}`)

## 4. Variables (no-secret) en GitHub

**Settings → Secrets and variables → Actions → Variables → New repository variable**:

| Name | Valor |
|---|---|
| `GCP_PROJECT_ID` | `fidel-495520` |
| `GCP_REGION` | `us-central1` |
| `GCS_BUCKET` | `fidel-mvp-invoice` |
| `WHATSAPP_PHONE_NUMBER_ID` | `909785642227873` |
| `WHATSAPP_DISPLAY_PHONE` | `+15551857769` |

## 5. Cargar `DATABASE_URL` como secret en Secret Manager

El workflow lee `DATABASE_URL` de Secret Manager (en lugar de hardcodearla). Carga una vez:

```bash
DB_PASSWORD=$(gcloud secrets versions access latest --secret=DB_PASSWORD)
DATABASE_URL="postgresql://loyalty:${DB_PASSWORD}@/loyalty?host=/cloudsql/${PROJECT_ID}:${REGION}:${DB_INSTANCE}"
echo -n "$DATABASE_URL" | gcloud secrets create DATABASE_URL --data-file=- || \
  echo -n "$DATABASE_URL" | gcloud secrets versions add DATABASE_URL --data-file=-

gcloud secrets add-iam-policy-binding DATABASE_URL \
  --member="serviceAccount:fidel-quick-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor" \
  --quiet
```

## 6. Verificar el primer run

Después de un `git push origin main`:

1. https://github.com/fidelwa/fidel-quick/actions
2. Click en el workflow `deploy` que se disparó.
3. Esperar ~5-7 min (build) + ~1-2 min (deploy).
4. Al final aparece `🚀 Deployed to: https://fidel-quick-...run.app` y un smoke `curl /healthz`.

## Rollback

Si un deploy rompe algo:

```bash
# Listar revisiones
gcloud run revisions list --service=fidel-quick --region=us-central1

# Volver a la anterior
gcloud run services update-traffic fidel-quick \
  --to-revisions=fidel-quick-00042-xyz=100 \
  --region=us-central1
```

## Rotar la SA key

Recomendado cada 90 días:

```bash
# Lista keys
gcloud iam service-accounts keys list --iam-account="$CI_SA"

# Crea nueva, sube a GitHub Secrets, después borra la vieja
gcloud iam service-accounts keys delete <KEY_ID> --iam-account="$CI_SA"
```

## Anti-pattern

- ❌ NO commitear el JSON de la SA al repo. `.gitignore` debe cubrir `*.json` que tenga credenciales (revisar antes de cada commit con `git diff --cached`).
- ❌ NO dar `roles/owner` al SA — solo los roles mínimos listados.
- ❌ NO usar la misma SA del Cloud Run runtime (`fidel-quick-sa`) para CI — separación de privilegios.
