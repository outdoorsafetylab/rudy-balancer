# The values live here, not in a terraform.tfvars: there is one environment, and every input
# must be in git so the plan on a pull request is the apply that follows. A local
# terraform.tfvars would silently override them; do not keep one.
variable "project" {
  description = "The GCP project id."
  type        = string
  default     = "outdoorsafetylab"
}

variable "region" {
  type    = string
  default = "asia-east1"
}

variable "healthcheck_job" {
  description = "false until the StatusPage key has been added as a version of the statuspage-key secret (deploy/terraform/README.md): the job reads that secret, so it is declared, and scheduled, only once there is a value to read. Turned on 2026-10-18, once version 1 was added."
  type        = bool
  default     = true
}

variable "healthcheck_scheduled" {
  description = "Whether Cloud Scheduler starts this project's health check. Turned off at the cutover of rudymap.tw (#32 plan D9), in the same change that turns rudymap's on: one health check runs at a time. The job itself stays, for a rollback, until the module is emptied."
  type        = bool
  default     = false
}

variable "healthcheck_schedule" {
  description = "How often Cloud Scheduler starts the health check, as cron in Asia/Taipei. A run takes about five minutes, most of it waiting on mirrors that time out."
  type        = string
  default     = "*/30 * * * *"
}
