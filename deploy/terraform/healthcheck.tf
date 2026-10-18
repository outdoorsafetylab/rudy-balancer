# The mirror health check (`rudy-balancer healthcheck -c docker`) as a Cloud Run job started by
# Cloud Scheduler, as in deploy/terraform-outdoorsafetylab/. The job is declared once this
# project's registry holds an image (healthcheck_image); the schedule only from the cutover of
# rudymap.tw (healthcheck_scheduled, plan D9). Production deploys swap the job's image
# (cloudbuild.yaml).

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

resource "google_secret_manager_secret_iam_member" "healthcheck_statuspage_key" {
  secret_id = google_secret_manager_secret.balancer["statuspage-key"].id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.healthcheck.member
}

resource "google_cloud_run_v2_job" "healthcheck" {
  count    = var.healthcheck_image != "" ? 1 : 0
  name     = "rudy-healthcheck"
  location = var.region

  template {
    task_count = 1
    template {
      service_account = google_service_account.healthcheck.email
      timeout         = "900s"
      max_retries     = 0
      containers {
        # Only the first create uses this: cloudbuild.yaml replaces it on every production deploy.
        image   = var.healthcheck_image
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
              secret  = google_secret_manager_secret.balancer["statuspage-key"].secret_id
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
    google_project_service.balancer,
    google_secret_manager_secret_iam_member.healthcheck_statuspage_key,
  ]
}

# Starting the job is run.jobs.run, which the invoker role carries.
resource "google_cloud_run_v2_job_iam_member" "healthcheck_runs_itself" {
  count    = length(google_cloud_run_v2_job.healthcheck)
  location = var.region
  name     = google_cloud_run_v2_job.healthcheck[0].name
  role     = "roles/run.invoker"
  member   = google_service_account.healthcheck.member
}

# Starting a job is a call to the Cloud Run Admin API, which takes an OAuth access token; nothing
# secret is stored in the job resource. No retry_config: a missed start is made up by the next
# one, zero retries is the default, and the API stores an all-zero retry config as absent, so
# declaring one shows as a change in every plan.
resource "google_cloud_scheduler_job" "healthcheck" {
  count     = var.healthcheck_scheduled && var.healthcheck_image != "" ? 1 : 0
  name      = "rudy-healthcheck"
  region    = var.region
  schedule  = var.healthcheck_schedule
  time_zone = "Asia/Taipei"

  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/projects/${var.project}/locations/${var.region}/jobs/${google_cloud_run_v2_job.healthcheck[0].name}:run"
    oauth_token {
      service_account_email = google_service_account.healthcheck.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }

  depends_on = [
    google_project_service.balancer,
    google_cloud_run_v2_job_iam_member.healthcheck_runs_itself,
    google_service_account_iam_member.terraform_acts_as_healthcheck,
  ]
}
