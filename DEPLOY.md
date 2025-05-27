# How We Deploy the App

We use [ArgoCD](https://argo-cd.readthedocs.io/en/stable/) to manage the deployment of our application.

## Background

ArgoCD runs on our `us-west-2` EKS cluster and connects to additional EKS clusters in `us-east-1` and `eu-west-1` to manage the application across all regions.
The ArgoCD setup is defined in [Terraform](./terraform/configurations/aws/argocd.hcl).

We use an [ApplicationSet](https://argo-cd.readthedocs.io/en/stable/user-guide/applicationset/) to generate three ArgoCD Applications, one for each cluster.
The ApplicationSet configuration is defined in [Terraform](./terraform/configurations/aws/userclouds-services.hcl).

## Deployment Process

### Debug & Staging Environments

When a branch is pushed to one of the deployment branches for [staging](https://github.com/userclouds/userclouds/tree/deploy/staging) or [debug](https://github.com/userclouds/userclouds/tree/deploy/debug), the following steps occur:

1. A [GitHub Action Workflow](https://github.com/userclouds/userclouds/blob/deploy/staging/.github/workflows/build-uc-containers.yml) is triggered.
   This workflow builds container images and pushes them to Amazon ECR.
   Authorization to access [AWS ECR](./terraform/modules/aws/ecr/uc-ecr-repos/github-actions-iam/) is handled using OIDC between GitHub and AWS.

2. The [ArgoCD Image Updater](https://argocd-image-updater.readthedocs.io/en/stable/) polls the AWS ECR repository for new images.
   - It primarily monitors the LogServer image, as images for other services share the same tag.
   - The LogServer image is always pushed last, ensuring that all other images are already in their respective ECR repositories.
   - The ArgoCD Image Updater then updates the ArgoCD Application with the new image tag.
   - ArgoCD deploys the new image to the clusters.

### Production Deployment

We don't want to build new container images for production; instead, we use the same images built and deployed for staging.

When new images are pushed to the staging ECR repositories, they are [automatically replicated](./terraform/configurations/aws/ecr-repos.hcl) to the production ECR repositories.
This is managed in Terraform: [ecr-repos.hcl:ecr_repos_internal_replication](./terraform/configurations/aws/ecr-repos.hcl).

To avoid triggering a deploy to production when the images become available (which happens when we push to staging), we set the ArgoCD ApplicationSet (and thus the ArgoCD Applications) [for production](./terraform/modules/userclouds/applications/userclouds/userclouds-appset.tftpl.yaml) to `syncPolicy: none`.
This means ArgoCD doesn't automatically sync the application when the image is updated (or any other changes occur).
The image update will still happen when the new images are available, but it won't sync (i.e., it won't change anything in the production clusters).
When we run the deploy process for production (`make deploy-prod`), the [underlying script](./tools/deploy.sh) [calls the ArgoCD CLI](./tools/deploy-with-argocd.sh) for each application (i.e., each cluster) to sync the application.
