# rudy-balancer's cloud resources in the rudymap project, where rudy-tiles also runs: one root
# module for the one project (#32, docs/proposal/move_to_rudymap_plan.md D2). Agents only write
# .tf; merging to master applies it (.github/workflows/terraform-*.yml). The Cloud Run services and
# the image are deployed by Cloud Build (cloudbuild.yaml), and the triggers are made by hand.
#
# rudy-tiles' own resources in this project (its registry, trigger, bucket, alerts) were made by
# hand and are not adopted here yet.
terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.30"
    }
  }

  backend "gcs" {
    # Created once by hand before the first apply (README.md); the name is fixed here because a
    # backend block cannot read variables.
    bucket = "rudy-balancer-tfstate-rudymap"
    prefix = "rudy-balancer"
  }
}

provider "google" {
  project = var.project
  region  = var.region
}
