# rudy-balancer's cloud resources in the outdoorsafetylab project: one root module for the one
# project. Agents only write .tf; merging to master applies it (.github/workflows/terraform-*.yml).
# The Cloud Run services and the image are still deployed by Cloud Build on a v* tag
# (cloudbuild.yaml), not here.
#
# First residents: the GitOps identity (gitops.tf) and the scheduled health check
# (healthcheck.tf). The project's older resources are not adopted.
#
# Being retired: the balancer moves to the rudymap project (#32,
# docs/proposal/move_to_rudymap_plan.md D2), whose module is deploy/terraform/. This one keeps the
# old project's health check running until the cutover, and is emptied and removed at the end.
terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.30"
    }
  }

  backend "gcs" {
    # Created once by hand before the first apply (deploy/terraform/README.md); the name is fixed
    # here because a backend block cannot read variables.
    bucket = "rudy-balancer-tfstate-outdoorsafetylab"
    prefix = "rudy-balancer"
  }
}

provider "google" {
  project = var.project
  region  = var.region
}
