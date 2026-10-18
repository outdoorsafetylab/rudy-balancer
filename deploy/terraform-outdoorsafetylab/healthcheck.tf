# The mirror health check (`rudy-balancer healthcheck -c docker`) as a Cloud Run job started by
# Cloud Scheduler. It replaced a scheduled GitHub workflow, which GitHub ran hours late and which
# needed a key file.
#
# What the job runs is the deploy's: cloudbuild.yaml swaps its image for the digest production
# just deployed, on the trigger that sets _HEALTHCHECK_JOB. Everything else about the job is here.

resource "google_project_service" "healthcheck" {
  for_each = toset([
    "run.googleapis.com",
    "cloudscheduler.googleapis.com",
    "secretmanager.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

resource "google_service_account" "healthcheck" {
  account_id   = "rudy-healthcheck"
  display_name = "rudy-healthcheck"
  description  = "Runs the rudy-balancer health check job, and is the identity Cloud Scheduler starts it with."
}

# The check records each mirror's state in Firestore, which the balancer routes by.
resource "google_project_iam_member" "healthcheck_firestore" {
  project = var.project
  role    = "roles/datastore.user"
  member  = google_service_account.healthcheck.member
}

# The StatusPage API key. Only the container is Terraform's; the value is added as a version by
# hand (deploy/terraform/README.md) and never enters state.
resource "google_secret_manager_secret" "statuspage_key" {
  secret_id = "statuspage-key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.healthcheck]
}

resource "google_secret_manager_secret_iam_member" "healthcheck_statuspage_key" {
  secret_id = google_secret_manager_secret.statuspage_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.healthcheck.member
}

resource "google_cloud_run_v2_job" "healthcheck" {
  count    = var.healthcheck_job ? 1 : 0
  name     = "rudy-healthcheck"
  location = var.region

  template {
    task_count = 1
    template {
      service_account = google_service_account.healthcheck.email
      # A run takes about five minutes; GitHub's job had a fifteen-minute limit.
      timeout     = "900s"
      max_retries = 0
      containers {
        # Only the first create uses this: cloudbuild.yaml replaces it on every production deploy.
        # It is the digest production served on 2026-10-18, not the tag, which the alpha build on
        # master also pushes.
        image   = "asia-northeast1-docker.pkg.dev/outdoorsafetylab/rudy-balancer/rudy-balancer@sha256:43603fb0def18c059c7c9ec403628b8bf9595c8aa7d77410e62cdfeec97ba99a"
        command = ["/usr/sbin/rudy-balancer"]
        args    = ["healthcheck", "-c", "docker"]
        resources {
          limits = {
            cpu    = "1"
            memory = "512Mi"
          }
        }
        env {
          name = "STATUSPAGE_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.statuspage_key.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  lifecycle {
    # The image is the deploy's; client and client_version are stamped by its `gcloud run jobs
    # update`.
    ignore_changes = [template[0].template[0].containers[0].image, client, client_version]
  }

  depends_on = [
    google_project_service.healthcheck,
    google_secret_manager_secret_iam_member.healthcheck_statuspage_key,
  ]
}

# Starting the job is run.jobs.run, which the invoker role carries.
resource "google_cloud_run_v2_job_iam_member" "healthcheck_runs_itself" {
  count    = var.healthcheck_job ? 1 : 0
  location = var.region
  name     = google_cloud_run_v2_job.healthcheck[0].name
  role     = "roles/run.invoker"
  member   = google_service_account.healthcheck.member
}

# Starting a job is a call to the Cloud Run Admin API, which takes an OAuth access token, not the
# ID token a call to a service of ours would; neither puts a secret in the job resource.
resource "google_cloud_scheduler_job" "healthcheck" {
  count     = var.healthcheck_job ? 1 : 0
  name      = "rudy-healthcheck"
  region    = var.region
  schedule  = var.healthcheck_schedule
  time_zone = "Asia/Taipei"

  # No retry_config: a missed start is made up by the next one, and zero retries is the default.
  # Declaring retry_count = 0 left the plan showing a change forever, because the API stores an
  # all-zero retry config as absent.

  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/projects/${var.project}/locations/${var.region}/jobs/${google_cloud_run_v2_job.healthcheck[0].name}:run"
    oauth_token {
      service_account_email = google_service_account.healthcheck.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }

  depends_on = [
    google_project_service.healthcheck,
    google_cloud_run_v2_job_iam_member.healthcheck_runs_itself,
    google_service_account_iam_member.terraform_acts_as_healthcheck,
  ]
}
