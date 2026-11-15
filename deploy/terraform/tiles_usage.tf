# The tiles service's weekly usage export: its build counts the past days' requests by client from
# Cloudflare's analytics and keeps them here as CSV, since Cloudflare holds them for only about a
# month. The build, its identity and that identity's access to the two resources below belong to
# the tiles service and are made by hand like its triggers; this file declares only where the files
# go and the token the build reads.

# Kept, not cached: no lifecycle rule deletes the files, and versioning keeps an overwritten one.
resource "google_storage_bucket" "tiles_usage" {
  name                        = "${var.project}-tiles-usage"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  versioning {
    enabled = true
  }
  depends_on = [google_project_iam_member.terraform]
}

# A Cloudflare API token that can read the zone's analytics and nothing else. The container only;
# the value is added as a version by hand and never enters state.
resource "google_secret_manager_secret" "cloudflare_analytics" {
  secret_id = "cloudflare-analytics-token"
  replication {
    auto {}
  }
  depends_on = [google_project_service.balancer]
}
