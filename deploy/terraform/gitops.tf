# Terraform GitOps: GitHub Actions plans on pull requests and applies on master, authenticating
# through Workload Identity Federation, so there is no key file. The pool, the provider, the
# service account and its roles are themselves declared here: the first apply is run by hand
# (README.md), and from then on the workflow applies everything, this file included.

locals {
  github_repository = "outdoorsafetylab/rudy-balancer"
}

# ---------------------------------------------------------------- APIs the exchange needs

# The token exchange (sts), the impersonation behind it (iamcredentials) and the project read
# every plan starts with (cloudresourcemanager). Never disabled on destroy: other things use them.
resource "google_project_service" "gitops" {
  for_each = toset([
    "sts.googleapis.com",
    "iamcredentials.googleapis.com",
    "cloudresourcemanager.googleapis.com",
  ])
  service            = each.value
  disable_on_destroy = false
}

# ---------------------------------------------------------------- identity

# .github/workflows/terraform-*.yml carry the provider's resource name and the SA's email as
# constants:
#   projects/933372650185/locations/global/workloadIdentityPools/github-actions/providers/rudy-balancer
#   rudy-balancer-terraform@rudymap.iam.gserviceaccount.com
resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"
  depends_on                = [google_project_service.gitops]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "rudy-balancer"
  display_name                       = "rudy-balancer"

  # Only this repository's workflows get a token out of the pool, whatever they ask for. The
  # repository is public: a pull request from a fork gets no OIDC token from GitHub, so only
  # branches pushed by someone with write access can plan.
  attribute_condition = "assertion.repository == \"${local.github_repository}\""

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
  }

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "terraform" {
  account_id   = "rudy-balancer-terraform"
  display_name = "rudy-balancer-terraform"
  description  = "Terraform GitOps for outdoorsafetylab/rudy-balancer: plans on pull requests, applies on master. Holds one role per kind of resource deploy/terraform/ declares."
}

# Every workflow of the repository may run as the SA. Whether a run plans or applies is decided
# by the workflow files (plan on pull_request, apply on push to master), not by a second identity.
resource "google_service_account_iam_member" "terraform_github" {
  service_account_id = google_service_account.terraform.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${local.github_repository}"
}

# ---------------------------------------------------------------- what the SA may touch

# One role per kind of resource this module declares, no more: a resource kind that arrives later
# adds its role here in the same PR. projectIamAdmin reaches furthest — it lets this file grant the
# roles below, and would let a merged .tf grant anything else in the project; what keeps it in
# bounds is the repository condition above and that the workflow only applies what is merged.
locals {
  terraform_roles = [
    "roles/serviceusage.serviceUsageAdmin",  # google_project_service
    "roles/iam.serviceAccountAdmin",         # the service accounts and their IAM
    "roles/iam.workloadIdentityPoolAdmin",   # this file's pool and provider
    "roles/resourcemanager.projectIamAdmin", # google_project_iam_member; reads the project number
    "roles/datastore.owner",                 # the Firestore database
    "roles/artifactregistry.admin",          # the image repository and who may push to it
    "roles/bigquery.dataOwner",              # the redirect log dataset and who may write to it
    "roles/logging.configWriter",            # the log sink that fills it
    "roles/secretmanager.admin",             # the secret containers and who may read them
    "roles/run.admin",                       # the health check job and who may start it
    "roles/cloudscheduler.admin",            # the job's schedule
  ]
}

resource "google_project_iam_member" "terraform" {
  for_each = toset(local.terraform_roles)
  project  = var.project
  role     = each.value
  member   = google_service_account.terraform.member
}

# The state bucket, created by hand and kept outside the module so a mistaken destroy can never
# take the state with it. storage.admin, scoped to this bucket: the state needs object access, and
# this very binding needs the bucket's getIamPolicy and setIamPolicy, which no object role carries.
resource "google_storage_bucket_iam_member" "terraform_state" {
  bucket = "rudy-balancer-tfstate-${var.project}"
  role   = "roles/storage.admin"
  member = google_service_account.terraform.member
}

# The job runs as rudy-healthcheck and the schedule starts it with that SA's token; declaring
# either takes being allowed to act as the SA.
resource "google_service_account_iam_member" "terraform_acts_as_healthcheck" {
  service_account_id = google_service_account.healthcheck.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.terraform.member
}
