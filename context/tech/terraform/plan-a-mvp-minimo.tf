# =============================================================================
# Plan A: MVP Minimo — ~$9/mes
# =============================================================================
#
# Cloud Run (free tier, scale to zero) + Cloud SQL db-f1-micro + Upstash Redis (externo)
# Sin VPC connector, sin Memorystore
# Ideal para validar con 1-5 negocios, <500 usuarios
#
# Uso:
#   cp plan-a-mvp-minimo.tf main.tf
#   terraform init
#   terraform plan
#   terraform apply
#
# =============================================================================

terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

# -----------------------------------------------------------------------------
# Variables
# -----------------------------------------------------------------------------

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "db_password" {
  description = "PostgreSQL password for loyalty user"
  type        = string
  sensitive   = true
}

variable "upstash_redis_url" {
  description = "Upstash Redis URL (external)"
  type        = string
  sensitive   = true
}

variable "whatsapp_api_token" {
  description = "WhatsApp Business API token"
  type        = string
  sensitive   = true
}

variable "whatsapp_verify_token" {
  description = "WhatsApp webhook verify token"
  type        = string
  sensitive   = true
}

variable "whatsapp_phone_number_id" {
  description = "WhatsApp phone number ID"
  type        = string
}

variable "whatsapp_display_phone" {
  description = "WhatsApp display phone number"
  type        = string
  default     = ""
}

variable "jwt_secret" {
  description = "JWT signing secret"
  type        = string
  sensitive   = true
}

variable "bearer_token" {
  description = "API Bearer token"
  type        = string
  sensitive   = true
}

variable "anthropic_api_key" {
  description = "Anthropic API key for OCR"
  type        = string
  sensitive   = true
}

variable "container_image" {
  description = "Docker image URL (e.g. us-central1-docker.pkg.dev/project/repo/fidel-quick:latest)"
  type        = string
}

# -----------------------------------------------------------------------------
# Provider
# -----------------------------------------------------------------------------

provider "google" {
  project = var.project_id
  region  = var.region
}

# -----------------------------------------------------------------------------
# APIs
# -----------------------------------------------------------------------------

resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "sqladmin.googleapis.com",
    "secretmanager.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
  ])

  service            = each.value
  disable_on_destroy = false
}

# -----------------------------------------------------------------------------
# Artifact Registry
# -----------------------------------------------------------------------------

resource "google_artifact_registry_repository" "repo" {
  location      = var.region
  repository_id = "fidel-repo"
  format        = "DOCKER"

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# Cloud SQL — PostgreSQL (db-f1-micro)
# -----------------------------------------------------------------------------

resource "google_sql_database_instance" "postgres" {
  name             = "fidel-db"
  database_version = "POSTGRES_16"
  region           = var.region

  settings {
    tier              = "db-f1-micro"
    availability_type = "ZONAL"
    disk_size         = 10
    disk_type         = "PD_SSD"

    backup_configuration {
      enabled    = true
      start_time = "03:00"
    }

    ip_configuration {
      ipv4_enabled = false
    }
  }

  deletion_protection = true

  depends_on = [google_project_service.apis]
}

resource "google_sql_database" "loyalty" {
  name     = "loyalty"
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_user" "loyalty" {
  name     = "loyalty"
  instance = google_sql_database_instance.postgres.name
  password = var.db_password
}

# -----------------------------------------------------------------------------
# Cloud Storage — Fotos de tickets
# -----------------------------------------------------------------------------

resource "google_storage_bucket" "invoices" {
  name     = "${var.project_id}-loyalty-invoices"
  location = var.region

  uniform_bucket_level_access = true

  lifecycle_rule {
    condition {
      age = 365
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }
}

# -----------------------------------------------------------------------------
# Secret Manager
# -----------------------------------------------------------------------------

locals {
  secrets = {
    WHATSAPP_API_TOKEN    = var.whatsapp_api_token
    WHATSAPP_VERIFY_TOKEN = var.whatsapp_verify_token
    JWT_SECRET            = var.jwt_secret
    BEARER_TOKEN          = var.bearer_token
    ANTHROPIC_API_KEY     = var.anthropic_api_key
  }
}

resource "google_secret_manager_secret" "secrets" {
  for_each  = local.secrets
  secret_id = each.key

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "secrets" {
  for_each    = local.secrets
  secret      = google_secret_manager_secret.secrets[each.key].id
  secret_data = each.value
}

# -----------------------------------------------------------------------------
# Service Account for Cloud Run
# -----------------------------------------------------------------------------

resource "google_service_account" "cloudrun" {
  account_id   = "fidel-quick-sa"
  display_name = "fidel-quick Cloud Run"
}

resource "google_project_iam_member" "cloudrun_sql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.cloudrun.email}"
}

resource "google_project_iam_member" "cloudrun_storage" {
  project = var.project_id
  role    = "roles/storage.objectUser"
  member  = "serviceAccount:${google_service_account.cloudrun.email}"
}

resource "google_secret_manager_secret_iam_member" "cloudrun_secrets" {
  for_each  = google_secret_manager_secret.secrets
  secret_id = each.value.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloudrun.email}"
}

# -----------------------------------------------------------------------------
# Cloud Run — API + Admin
# -----------------------------------------------------------------------------

resource "google_cloud_run_v2_service" "api" {
  name     = "fidel-quick"
  location = var.region

  template {
    service_account = google_service_account.cloudrun.email

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = var.container_image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          memory = "256Mi"
          cpu    = "1"
        }
      }

      # Environment variables
      env {
        name  = "ENV"
        value = "production"
      }
      env {
        name  = "PORT"
        value = "8080"
      }
      env {
        name  = "S3_BUCKET"
        value = google_storage_bucket.invoices.name
      }
      env {
        name  = "S3_REGION"
        value = var.region
      }
      env {
        name  = "REDIS_URL"
        value = var.upstash_redis_url
      }
      env {
        name  = "WHATSAPP_PHONE_NUMBER_ID"
        value = var.whatsapp_phone_number_id
      }
      env {
        name  = "WHATSAPP_DISPLAY_PHONE"
        value = var.whatsapp_display_phone
      }
      env {
        name  = "DATABASE_URL"
        value = "postgres://loyalty:${var.db_password}@/loyalty?host=/cloudsql/${google_sql_database_instance.postgres.connection_name}"
      }

      # Secrets
      dynamic "env" {
        for_each = local.secrets
        content {
          name = env.key
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.secrets[env.key].secret_id
              version = "latest"
            }
          }
        }
      }

      startup_probe {
        http_get {
          path = "/health"
        }
        initial_delay_seconds = 5
        period_seconds        = 10
      }
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }

    timeout = "60s"
  }

  depends_on = [
    google_secret_manager_secret_version.secrets,
    google_project_iam_member.cloudrun_sql,
  ]
}

# Allow unauthenticated access (WhatsApp webhook needs public access)
resource "google_cloud_run_v2_service_iam_member" "public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

output "api_url" {
  value       = google_cloud_run_v2_service.api.uri
  description = "Cloud Run service URL — usar como webhook URL en Meta"
}

output "cloud_sql_connection" {
  value       = google_sql_database_instance.postgres.connection_name
  description = "Cloud SQL connection name para Cloud SQL Proxy"
}

output "storage_bucket" {
  value       = google_storage_bucket.invoices.name
  description = "Cloud Storage bucket para fotos"
}

output "estimated_monthly_cost" {
  value       = "~$9/mes (Cloud SQL db-f1-micro $9 + Cloud Run free tier + Storage free tier)"
  description = "Costo estimado mensual"
}
