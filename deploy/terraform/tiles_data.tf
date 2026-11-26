# The tiles service's weekly sync of the vector edition to data.rudymap.tw: its release builds
# upload the week's files to the bucket behind it (data.tf) with a Cloudflare API token that can
# write that bucket's objects and nothing else. As for the usage export (tiles_usage.tf), the
# builds, their identity and that identity's access to the secret belong to the tiles service and
# are made by hand; this file declares only the token's container.

# The container only; the value is added as a version by hand and never enters state.
resource "google_secret_manager_secret" "tiles_data_writer" {
  secret_id = "r2-rudymap-data-writer"
  replication {
    auto {}
  }
  depends_on = [google_project_service.balancer]
}
