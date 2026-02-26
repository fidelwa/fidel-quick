# =============================================================================
# Plan B: GCP Standard — ~$55/mes
# =============================================================================
#
# Todo en GCP con recursos dedicados. Cloud Run (min=1, 512MB, cpu-boost)
# + Cloud SQL db-g1-small (1.7GB) + Memorystore Redis 1GB
# + VPC connector + Cloud Logging + Monitoring
# Listo para produccion, 20-100 negocios, <50,000 usuarios
#
# Uso:
#   cp plan-b-gcp-standard.tf main.tf
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

  # Recomendado: backend remoto para state compartido
  # backend "gcs" {
  #   bucket = "fidel-terraform-state"
  #   prefix = "production"
  # }
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

variable "platform_url" {
  description = "Platform URL for deeplinks"
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
  description = "Docker image URL"
  type        = string
}

variable "custom_domain" {
  description = "Custom domain (optional, e.g. api.fidel.app)"
  type        = string
  default     = ""
}

variable "alert_email" {
  description = "Email for monitoring alerts"
  type        = string
  default     = ""
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
    "redis.googleapis.com",
    "vpcaccess.googleapis.com",
    "monitoring.googleapis.com",
    "logging.googleapis.com",
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

  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 10
    }
  }

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# Cloud SQL — PostgreSQL (db-g1-small, 1.7GB RAM)
# -----------------------------------------------------------------------------

resource "google_sql_database_instance" "postgres" {
  name             = "fidel-db"
  database_version = "POSTGRES_16"
  region           = var.region

  settings {
    tier              = "db-g1-small"
    availability_type = "ZONAL"
    disk_size         = 10
    disk_type         = "PD_SSD"
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
    }

    maintenance_window {
      day          = 7 # Sunday
      hour         = 4 # 4 AM
      update_track = "stable"
    }

    insights_config {
      query_insights_enabled  = true
      record_application_tags = true
      record_client_address   = false
    }

    ip_configuration {
      ipv4_enabled = false
    }

    database_flags {
      name  = "log_min_duration_statement"
      value = "1000" # Log queries > 1s
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
# Memorystore Redis — 1GB Basic
# -----------------------------------------------------------------------------

resource "google_redis_instance" "redis" {
  name               = "fidel-redis"
  tier               = "BASIC"
  memory_size_gb     = 1
  region             = var.region
  redis_version      = "REDIS_7_2"
  auth_enabled       = false
  transit_encryption_mode = "DISABLED" # VPC-only, no need for TLS overhead

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 4
        minutes = 0
      }
    }
  }

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# VPC Connector — Cloud Run -> Memorystore + Cloud SQL (private IP)
# -----------------------------------------------------------------------------

resource "google_vpc_access_connector" "connector" {
  name          = "fidel-connector"
  region        = var.region
  ip_cidr_range = "10.8.0.0/28"
  network       = "default"

  min_instances = 2
  max_instances = 3

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# Cloud Storage — Fotos de tickets
# -----------------------------------------------------------------------------

resource "google_storage_bucket" "invoices" {
  name     = "${var.project_id}-loyalty-invoices"
  location = var.region

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }

  lifecycle_rule {
    condition {
      age = 365
    }
    action {
      type          = "SetStorageClass"
      storage_class = "COLDLINE"
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

resource "google_project_iam_member" "cloudrun_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.cloudrun.email}"
}

resource "google_project_iam_member" "cloudrun_monitoring" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.cloudrun.email}"
}

resource "google_secret_manager_secret_iam_member" "cloudrun_secrets" {
  for_each  = google_secret_manager_secret.secrets
  secret_id = each.value.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloudrun.email}"
}

# -----------------------------------------------------------------------------
# Cloud Run — API + Admin (min=1, 512MB, cpu-boost)
# -----------------------------------------------------------------------------

resource "google_cloud_run_v2_service" "api" {
  name     = "fidel-quick"
  location = var.region

  template {
    service_account = google_service_account.cloudrun.email

    vpc_access {
      connector = google_vpc_access_connector.connector.id
      egress    = "PRIVATE_RANGES_ONLY"
    }

    scaling {
      min_instance_count = 1
      max_instance_count = 3
    }

    containers {
      image = var.container_image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          memory = "512Mi"
          cpu    = "1"
        }
        cpu_idle          = false # CPU always allocated (min=1)
        startup_cpu_boost = true
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
        value = "redis://${google_redis_instance.redis.host}:${google_redis_instance.redis.port}"
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
        name  = "PLATFORM_URL"
        value = var.platform_url
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
        initial_delay_seconds = 3
        period_seconds        = 5
        failure_threshold     = 3
      }

      liveness_probe {
        http_get {
          path = "/health"
        }
        period_seconds    = 30
        failure_threshold = 3
      }
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }

    timeout                          = "60s"
    max_instance_request_concurrency = 80
  }

  depends_on = [
    google_secret_manager_secret_version.secrets,
    google_project_iam_member.cloudrun_sql,
    google_vpc_access_connector.connector,
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
# Monitoring — Uptime check + Alert
# -----------------------------------------------------------------------------

resource "google_monitoring_uptime_check_config" "api_health" {
  display_name = "fidel-quick health check"
  timeout      = "10s"
  period       = "300s" # Every 5 minutes

  http_check {
    path         = "/health"
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = trimprefix(google_cloud_run_v2_service.api.uri, "https://")
    }
  }
}

resource "google_monitoring_notification_channel" "email" {
  count        = var.alert_email != "" ? 1 : 0
  display_name = "Fidel Alerts Email"
  type         = "email"
  labels = {
    email_address = var.alert_email
  }
}

resource "google_monitoring_alert_policy" "high_latency" {
  count        = var.alert_email != "" ? 1 : 0
  display_name = "fidel-quick: High Latency (>2s p95)"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run request latency > 2s"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"fidel-quick\" AND metric.type = \"run.googleapis.com/request_latencies\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 2000 # 2000ms

      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_PERCENTILE_95"
        cross_series_reducer = "REDUCE_MAX"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email[0].id]
}

resource "google_monitoring_alert_policy" "error_rate" {
  count        = var.alert_email != "" ? 1 : 0
  display_name = "fidel-quick: High Error Rate (>5%)"
  combiner     = "OR"

  conditions {
    display_name = "Cloud Run 5xx error rate > 5%"
    condition_threshold {
      filter          = "resource.type = \"cloud_run_revision\" AND resource.labels.service_name = \"fidel-quick\" AND metric.type = \"run.googleapis.com/request_count\" AND metric.labels.response_code_class = \"5xx\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 5

      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email[0].id]
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

output "redis_host" {
  value       = google_redis_instance.redis.host
  description = "Memorystore Redis host (solo accesible via VPC)"
}

output "redis_port" {
  value       = google_redis_instance.redis.port
  description = "Memorystore Redis port"
}

output "storage_bucket" {
  value       = google_storage_bucket.invoices.name
  description = "Cloud Storage bucket para fotos"
}

output "service_account" {
  value       = google_service_account.cloudrun.email
  description = "Service account del Cloud Run"
}

output "estimated_monthly_cost" {
  value       = "~$55/mes (Cloud Run min=1 512MB ~$15 + Cloud SQL db-g1-small $26+$2 + Memorystore 1GB $11 + Storage ~$0.20)"
  description = "Costo estimado mensual"
}
