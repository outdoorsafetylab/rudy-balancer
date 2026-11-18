# data.rudymap.tw: the vector map's files (PMTiles, styles, sprites, glyphs, DEM, terrain), served
# from R2 through Cloudflare. Paths are versioned (/<date>-<commit>/...) and never overwritten,
# except style.json and paper.json, which a release may replace within its version; the manifest
# that names the current version is https://tiles.rudymap.tw/pmtiles/latest.json, served by the
# tiles service. Objects carry their own Cache-Control, set when they are uploaded; the cache rule
# below makes every file eligible and keeps that header as the TTL, since Cloudflare by default
# caches neither .pmtiles nor .json. Uploading is not Terraform's: only the bucket and how it is
# served are declared here.

data "cloudflare_zone" "rudymap" {
  filter = {
    name = "rudymap.tw"
  }
}

data "cloudflare_rulesets" "rudymap" {
  zone_id = data.cloudflare_zone.rudymap.id
}

locals {
  cloudflare_account_id = data.cloudflare_zone.rudymap.account.id
  cache_ruleset_id      = one([for r in data.cloudflare_rulesets.rudymap.rulesets : r.id if r.phase == "http_request_cache_settings"])
}

resource "cloudflare_r2_bucket" "data" {
  account_id = local.cloudflare_account_id
  name       = "rudymap-data"
  location   = "apac"
}

resource "cloudflare_r2_custom_domain" "data" {
  account_id  = local.cloudflare_account_id
  bucket_name = cloudflare_r2_bucket.data.name
  domain      = "data.rudymap.tw"
  zone_id     = data.cloudflare_zone.rudymap.id
  enabled     = true
  min_tls     = "1.2"
}

# The zone's cache rules, all of them: a phase entrypoint ruleset is one object, so declaring it
# replaces every rule in it. The first rule was made by hand for the tiles service's place index and
# is kept exactly as it was; the import below adopts the ruleset, and the plan adds the other two.
# The manifest is served by the tiles service with its own Cache-Control (five minutes).
resource "cloudflare_ruleset" "cache" {
  zone_id = data.cloudflare_zone.rudymap.id
  name    = "default"
  kind    = "zone"
  phase   = "http_request_cache_settings"

  rules = [
    {
      description = "tiles POI index"
      expression  = "(http.host eq \"tiles.rudymap.tw\" and starts_with(http.request.uri.path, \"/poi/\"))"
      action      = "set_cache_settings"
      action_parameters = {
        cache       = true
        browser_ttl = { mode = "respect_origin" }
        edge_ttl    = { mode = "bypass_by_default" }
      }
      enabled = true
    },
    {
      description = "tiles vector manifest"
      expression  = "(http.host eq \"tiles.rudymap.tw\" and http.request.uri.path eq \"/pmtiles/latest.json\")"
      action      = "set_cache_settings"
      action_parameters = {
        cache       = true
        browser_ttl = { mode = "respect_origin" }
        edge_ttl    = { mode = "bypass_by_default" }
      }
      enabled = true
    },
    {
      description = "data.rudymap.tw"
      expression  = "(http.host eq \"data.rudymap.tw\")"
      action      = "set_cache_settings"
      action_parameters = {
        cache       = true
        browser_ttl = { mode = "respect_origin" }
        edge_ttl    = { mode = "respect_origin" }
      }
      enabled = true
    },
  ]
}

import {
  to = cloudflare_ruleset.cache
  id = "zones/${data.cloudflare_zone.rudymap.id}/${local.cache_ruleset_id}"
}
