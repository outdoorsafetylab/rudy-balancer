# deploy/terraform

The outdoorsafetylab project's resources that rudy-balancer needs, as one Terraform root module:
the GitOps identity (`gitops.tf`) and the scheduled health check (`healthcheck.tf`). The Cloud Run
services and the image stay with Cloud Build (`cloudbuild.yaml`).

## How a change goes live

A pull request that touches this directory gets a `terraform plan` comment
(`.github/workflows/terraform-plan.yml`), rewritten on every push. Merging it applies the merged
code (`.github/workflows/terraform-apply.yml`), so read the plan comment before merging: merging is
the irreversible step. Nobody runs `apply` by hand except to bootstrap and when the workflow is
broken; after such an apply, the next plan comment must read *No changes*.

## Bootstrap (once)

1. Create the state bucket, the one resource outside the module:

   ```sh
   gcloud storage buckets create gs://rudy-balancer-tfstate-outdoorsafetylab \
     --project=outdoorsafetylab --location=asia-east1 \
     --uniform-bucket-level-access --public-access-prevention
   gcloud storage buckets update gs://rudy-balancer-tfstate-outdoorsafetylab --versioning
   ```

2. Apply once by hand, as a person who can grant IAM in the project. This creates the GitOps
   identity the workflows use, the health check's service account and the empty `statuspage-key`
   secret. The job itself waits for `healthcheck_job` (step 4).

   ```sh
   cd deploy/terraform && terraform init && terraform apply
   ```

3. Add the StatusPage key as the secret's first version. It never enters git or state:

   ```sh
   printf '%s' "$KEY" | gcloud secrets versions add statuspage-key \
     --project=outdoorsafetylab --data-file=-
   ```

4. A pull request sets `healthcheck_job` to `true`; merging it creates the job and its schedule.

5. Let production deploys update the job's image:

   ```sh
   gcloud builds triggers update github rudymap --project=outdoorsafetylab --region=global \
     --update-substitutions=_HEALTHCHECK_JOB=rudy-healthcheck
   ```
