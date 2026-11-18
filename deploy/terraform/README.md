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
are made by hand. `deploy/terraform-outdoorsafetylab/` is the old project's module, kept until the
move is done.

## How a change goes live

A pull request that touches either module gets one `terraform plan` comment per module
(`.github/workflows/terraform-plan.yml`), rewritten on every push. Merging it applies the merged code
(`.github/workflows/terraform-apply.yml`), so read the plan comments before merging: merging is the
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

4. `data.tf` reads two Cloudflare tokens, neither in git or state:
   - plan (pull requests, which run their own code) reads a read-only one,
     the repository secret `CLOUDFLARE_API_TOKEN_READONLY`: R2 read on the
     account; zone read and cache settings read on rudymap.tw.
   - apply reads one that can write, the secret `CLOUDFLARE_API_TOKEN` of the
     `production` environment, whose deployment branches are master only:
     R2 write on the account; zone read and cache settings write on
     rudymap.tw. DNS write is added only while a custom domain is changed.

   ```sh
   gh secret set CLOUDFLARE_API_TOKEN_READONLY   # from this checkout
   gh secret set CLOUDFLARE_API_TOKEN --env production
   ```

The health check job waits for `healthcheck_image` and its schedule for `healthcheck_scheduled`
(plan steps 3 and 5).
