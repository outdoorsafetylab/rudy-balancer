# deploy/terraform

The rudymap project's resources that rudy-balancer needs, as one Terraform root module
(#32, `docs/proposal/move_to_rudymap_plan.md`):

- `gitops.tf`: the GitOps identity.
- `balancer.tf`: Firestore, the image repository, the service and build identities, the secret
  containers and the redirect log dataset.
- `healthcheck.tf`: the health check job and its schedule.
- `tiles_usage.tf`: where the tiles service's weekly build keeps its usage export, and the Cloudflare
  token it reads.
- `data.tf`: on Cloudflare, not in the project: the R2 bucket behind
  `data.rudymap.tw` (the vector map's files), its custom domain, and
  rudymap.tw's cache rules.

The Cloud Run services and the image stay with Cloud Build (`cloudbuild.yaml`), and the triggers
are made by hand.

## How a change goes live

A pull request that touches this module gets a `terraform plan` comment
(`.github/workflows/terraform-plan.yml`), rewritten on every push. Merging it applies the merged code
(`.github/workflows/terraform-apply.yml`), so read the plan comment before merging: merging is the
irreversible step. Nobody runs `apply` by hand except to bootstrap and when the workflow is broken;
after such an apply, the next plan comment must read *No changes*.

## Bootstrap (once)

1. Create the state bucket, the one resource outside the module:

   ```sh
   gcloud storage buckets create gs://rudy-balancer-tfstate-rudymap \
     --project=rudymap --location=asia-east1 \
     --uniform-bucket-level-access --public-access-prevention
   gcloud storage buckets update gs://rudy-balancer-tfstate-rudymap --versioning
   ```

2. Apply once by hand, as a person who can grant IAM in the project:

   ```sh
   cd deploy/terraform && terraform init && terraform apply
   ```

3. Add each secret's first version. None of the values enters git or state:

   ```sh
   printf '%s' "$VALUE" | gcloud secrets versions add <statuspage-key|aws-access-key-id|aws-secret-access-key|cloudflare-analytics-token> \
     --project=rudymap --data-file=-
   ```

4. The Cloudflare token for `data.tf` is a repository secret,
   `CLOUDFLARE_API_TOKEN`, read by both workflows (scope: R2 on the account;
   DNS, zone read and cache settings on rudymap.tw):

   ```sh
   gh secret set CLOUDFLARE_API_TOKEN   # from this checkout
   ```

The health check job waits for `healthcheck_image` and its schedule for `healthcheck_scheduled`
(plan steps 3 and 5).
