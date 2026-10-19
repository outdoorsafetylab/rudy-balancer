# The values live here, not in a terraform.tfvars: there is one environment, and every input
# must be in git so the plan on a pull request is the apply that follows. A local
# terraform.tfvars would silently override them; do not keep one.
variable "project" {
  description = "The GCP project id."
  type        = string
  default     = "rudymap"
}

variable "region" {
  type    = string
  default = "asia-east1"
}

variable "healthcheck_image" {
  description = "The image the health check job is created with, by digest. Empty until this project's registry holds an image (plan step 2); the job is declared only once it is set, and production deploys replace it afterwards (cloudbuild.yaml)."
  type        = string
  # rubymap-00002-gdg, the first production revision in this project (built from master 5fec2ee).
  default = "asia-east1-docker.pkg.dev/rudymap/rudy-balancer/rudy-balancer@sha256:8d1f67b17fec7625880310372c0aad9b504ea559f840ef56753e05e821231e50"
}

variable "healthcheck_scheduled" {
  description = "Whether Cloud Scheduler starts the health check. Off until the cutover of rudymap.tw (plan D9): one health check runs at a time, and until then it is the outdoorsafetylab one."
  type        = bool
  default     = false
}

variable "healthcheck_schedule" {
  description = "How often Cloud Scheduler starts the health check, as cron in Asia/Taipei."
  type        = string
  default     = "*/30 * * * *"
}
