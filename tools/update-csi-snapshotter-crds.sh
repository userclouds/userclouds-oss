#!/bin/bash
set -euo pipefail

# https://github.com/kubernetes-sigs/aws-ebs-csi-driver/issues/1268
REPO_VERSION=v8.1.1
TARGET_PATH=./terraform/modules/aws/eks-cluster/software/crds/csi-snapshotter
BASE_URL=https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${REPO_VERSION}/client/config/crd
declare -a CRDS=(
  "volumesnapshotclasses"
  "volumesnapshotcontents"
  "volumesnapshots"

)

for crd in "${CRDS[@]}"; do
  crd_file="snapshot.storage.k8s.io_${crd}.yaml"
  target_file="${TARGET_PATH}/${crd_file}"
  echo "Downloading ${crd_file}..."
  if ! curl -f -s -S -o "${target_file}" "${BASE_URL}/${crd_file}"; then
    echo "Error downloading ${crd_file}" >&2
    exit 1
  fi
  # Remove status field from CRD, not sure why it is there in the first place.
  yq eval 'del(.status)' -i "$target_file"
done
echo "All CRDs downloaded successfully!"
