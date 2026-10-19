# What the balancer's services need in rudymap (docs/proposal/move_to_rudymap_plan.md step 1):
# the Firestore database, the image repository, the identities the services and their builds run
# as, the secret containers and the redirect log dataset. The services themselves are deployed by
# Cloud Build (cloudbuild.yaml); the triggers that run it are made by hand.

resource "google_project_service" "balancer" {
  for_each = toset([
    "firestore.googleapis.com",
    "artifactregistry.googleapis.com",
    "bigquery.googleapis.com",
    "run.googleapis.com",
    "cloudscheduler.googleapis.com",
    "secretmanager.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

# ---------------------------------------------------------------- data

# Plan D3: in the services' region; the old database is in asia-northeast2. The location cannot be
# changed once the database exists, and a destroy here leaves the database in place (ABANDON) with
# delete protection on.
resource "google_firestore_database" "default" {
  name                    = "(default)"
  location_id             = var.region
  type                    = "FIRESTORE_NATIVE"
  delete_protection_state = "DELETE_PROTECTION_ENABLED"
  deletion_policy         = "ABANDON"
  depends_on              = [google_project_service.balancer]
}

# Plan D5: the old project's history is merged in once, at the end.
resource "google_bigquery_dataset" "redirects" {
  dataset_id = "rudymap_file_redirect"
  location   = var.region
  depends_on = [google_project_service.balancer]
}

# Every redirect a service logs, into the dataset the quarterly report reads; the same filter and
# partitioning as the old project's sink, so the tables line up. No unique_writer_identity: in the
# sink's own project the provider reads it back as true and every plan would show a change.
resource "google_logging_project_sink" "redirects" {
  name        = "rudymap-file-redirect"
  destination = "bigquery.googleapis.com/projects/${var.project}/datasets/${google_bigquery_dataset.redirects.dataset_id}"
  filter      = <<-EOT
    resource.type = "cloud_run_revision"
    resource.labels.service_name = "rubymap" OR resource.labels.service_name = "ruby-balancer" OR resource.labels.service_name = "ruby-balancer-alpha"
    resource.labels.location = "asia-east1"
    jsonPayload.message =~ "^Redirecting"
    jsonPayload.File != ""
  EOT

  bigquery_options {
    use_partitioned_tables = true
  }

  # The apply that adds logging.configWriter to this module's own identity is the one that first
  # creates the sink.
  depends_on = [google_project_iam_member.terraform]
}

resource "google_bigquery_dataset_iam_member" "redirects_sink" {
  dataset_id = google_bigquery_dataset.redirects.dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = google_logging_project_sink.redirects.writer_identity
}

# ---------------------------------------------------------------- images

resource "google_artifact_registry_repository" "balancer" {
  repository_id = "rudy-balancer"
  location      = var.region
  format        = "DOCKER"
  depends_on    = [google_project_service.balancer]
}

# ---------------------------------------------------------------- identities

# The three services (rubymap, ruby-balancer, ruby-balancer-alpha) run as this.
resource "google_service_account" "balancer" {
  account_id   = "rudy-balancer"
  display_name = "rudy-balancer"
  description  = "Runs the rudy-balancer Cloud Run services."
}

resource "google_project_iam_member" "balancer_firestore" {
  project = var.project
  role    = "roles/datastore.user"
  member  = google_service_account.balancer.member
}

# The builds of the hand-made triggers run as this: push the image, deploy the services as
# rudy-balancer, and point the health check job at the new image.
resource "google_service_account" "builder" {
  account_id   = "rudy-balancer-builder"
  display_name = "rudy-balancer-builder"
  description  = "Runs the rudy-balancer Cloud Build triggers."
}

resource "google_artifact_registry_repository_iam_member" "builder_push" {
  location   = google_artifact_registry_repository.balancer.location
  repository = google_artifact_registry_repository.balancer.name
  role       = "roles/artifactregistry.writer"
  member     = google_service_account.builder.member
}

# run.admin, as rudy-tiles-builder has: a first deploy makes the service public, which is setting
# its IAM policy.
resource "google_project_iam_member" "builder" {
  for_each = toset([
    "roles/run.admin",
    "roles/logging.logWriter",
  ])
  project = var.project
  role    = each.value
  member  = google_service_account.builder.member
}

resource "google_service_account_iam_member" "builder_acts_as" {
  for_each = {
    balancer    = google_service_account.balancer.name
    healthcheck = google_service_account.healthcheck.name
  }
  service_account_id = each.value
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.builder.member
}

# ---------------------------------------------------------------- secrets

# Only the containers are Terraform's; each value is added as a version by hand and never enters
# state.
resource "google_secret_manager_secret" "balancer" {
  for_each  = toset(["statuspage-key", "aws-access-key-id", "aws-secret-access-key"])
  secret_id = each.value
  replication {
    auto {}
  }
  depends_on = [google_project_service.balancer]
}

resource "google_secret_manager_secret_iam_member" "balancer" {
  for_each  = google_secret_manager_secret.balancer
  secret_id = each.value.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.balancer.member
}
