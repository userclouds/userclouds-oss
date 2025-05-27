# Helm Chart for UserClouds Services

This Helm chart enables the deployment of UserClouds services on a Kubernetes cluster using the [Helm package manager](http://helm.sh).

## Overview

The chart is composed of templates written in the [Go template language](https://pkg.go.dev/text/template) with [Helm-specific extensions](https://helm.sh/docs/chart_template_guide/). These templates are rendered into Kubernetes manifest files and are located in the `templates` directory.

Configuration is managed through `values_xxxx.yaml` files, where `xxxx` denotes the environment or universe name. Additionally, per-region values files can be used alongside environment-specific files to support region-based customization.

For each UserClouds service, the chart provisions Kubernetes [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and [Service](https://kubernetes.io/docs/concepts/services-networking/service/) resources. An [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resource is also created: The Ingress resource utilizes AWS Application Load Balancers (ALBs) to route traffic appropriately and has annotations for the AWS ALB ingress controller. The chart also includes a [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) for configuration management.

Scheduled tasks are managed via [CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) defined in [worker-cronjobs.yaml](./templates/worker-cronjobs.yaml), which trigger relevant endpoints on the Worker service. For a list of endpoints, refer to [addCronEndpoints](./../../worker/cmd/main.go#L96).

## Configuration

UserClouds services (Log Server, Authz, Userstore, Plex, and Worker) are configured using a [ConfigMap](./templates/configs.yaml), which is mounted as a volume in their respective deployments. The ConfigMap aggregates configuration values from:

1. The primary `values.yaml` file
2. Base configuration files in the `./base_configs` directory

The base configuration files define common settings shared across all UserClouds services, ensuring a consistent baseline environment configuration.
