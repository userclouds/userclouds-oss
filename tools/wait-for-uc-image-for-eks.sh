#!/usr/bin/env bash

set -euo pipefail

source ./tools/packaging/helpers.sh
source ./tools/argocd-helpers.sh

IMAGE_TAG="$(current_version)"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --profile "$UC_UNIVERSE" --query Account --output text)
sp="/-\|" # Spinner
# It takes ~10m to build and push all images, so we'll wait for 15m (60 attempts * 15s)
MAX_ATTEMPTS=60
SLEEP_INTERVAL=15

ARGOCD_APP=$(get_argocd_apps | cut -d' ' -f1)

# Function to check if the image tag exists
check_image_tag() {
  local ecr_repo_name=$1
  local img_tag=$2
  local output
  output=$(aws ecr describe-images --profile "$UC_UNIVERSE" --registry-id "$AWS_ACCOUNT_ID" --repository-name "$ecr_repo_name" --no-cli-pager --region us-west-2 --query "imageDetails[?contains(imageTags, '$img_tag')]" --output text)
  if [ -n "$output" ]; then
    return 0
  else
    return 1
  fi
}

WORKFLOW_URL="https://github.com/userclouds/userclouds/actions/workflows/build-uc-container.yml?query=branch:deploy/${UC_UNIVERSE}"
echo "Workflow URL: $WORKFLOW_URL"
# Wait for the image tag to exist
attempt=0
REPOSITORY_NAME="userclouds"
echo -n "Checking for image tag '$IMAGE_TAG' in repository '$REPOSITORY_NAME'...  "
while [ $attempt -lt $MAX_ATTEMPTS ]; do
  if check_image_tag "$REPOSITORY_NAME" "$IMAGE_TAG"; then
    break
  fi
  # Make the UI "responsive" by printing a spinner, even though we only check evert 10 seconds (SLEEP_INTERVAL)
  for ((i = 1; i <= SLEEP_INTERVAL; i++)); do
    printf '\b%s' "${sp:i%${#sp}:1}"
    sleep 1
  done
done
echo
if [ $attempt -eq $MAX_ATTEMPTS ]; then
  echo "Image tag '$IMAGE_TAG' not found in repository '$REPOSITORY_NAME' after $((MAX_ATTEMPTS * SLEEP_INTERVAL)) seconds. Exiting..."
  exit 1
else
  echo "Image tag '$IMAGE_TAG' found in repository '$REPOSITORY_NAME'."
fi

echo "All image tags found in ECR repositories - ArgoCD should be updating apps see https://argocd.${UC_UNIVERSE}.userclouds.tools/applications"

ARGOCD_CLUSTER="${UC_UNIVERSE}-us-west-2"
# After the image is published to ECR, the ArgoCD Image Updater should detect that and update the ArgoCD Apps CRDs with the new image tag.
echo "Waiting for ArgoCD app $ARGOCD_APP to update to image tag $IMAGE_TAG..."
attempt=0
while [ $attempt -lt $MAX_ATTEMPTS ]; do
  current_tag=$(kubectl get app "$ARGOCD_APP" --context "$ARGOCD_CLUSTER" -n argocd -o jsonpath='{.spec.source.helm.parameters[?(@.name=="image.tag")].value}')
  if [ "$current_tag" = "$IMAGE_TAG" ]; then
    echo "ArgoCD app $ARGOCD_APP updated to image tag $IMAGE_TAG"
    break
  fi
  attempt=$((attempt + 1))
  # Show spinner while waiting
  for ((i = 1; i <= SLEEP_INTERVAL; i++)); do
    printf '\b%s' "${sp:i%${#sp}:1}"
    sleep 1
  done
done

if [ $attempt -eq $MAX_ATTEMPTS ]; then
  echo "Timeout waiting for ArgoCD app $ARGOCD_APP to update to image tag $IMAGE_TAG"
  exit 1
fi
