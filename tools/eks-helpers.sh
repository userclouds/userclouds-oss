#!/usr/bin/env bash

set -euo pipefail

# Function to check if a command is installed
check_command() {
  local cmd=$1
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: $cmd is not installed."
    echo "Please run: 'make install-tools'"
    exit 1
  fi
}

# Function to check if a cluster is configured and accessible
check_cluster() {
  local cluster_name=$1
  if ! kubectl config get-contexts | grep -qzF "$cluster_name"; then
    echo "Error: Kubernetes context for cluster '$cluster_name' is not configured."
    echo "run ./tools/ensure-kubectl-config.sh to configure the context."
    exit 1
  fi

  if ! kubectl get nodes --request-timeout 2 --context "$cluster_name" &>/dev/null; then
    echo "Error: Unable to access the Kubernetes cluster '$cluster_name'."
    echo "Make sure you are connected the the VPN for '$UC_UNIVERSE'."
    echo "To setup the VPN connection to the AWS Sub Account for each environment, see: https://www.notion.so/userclouds/VPN-setup-dcd214286f6541be8d1b4af461caf965?pvs=4#4e5c92a4730846db92396717aa9f7b39"
    exit 1
  fi

  echo "Kubernetes cluster '$cluster_name' is configured and accessible."
}
