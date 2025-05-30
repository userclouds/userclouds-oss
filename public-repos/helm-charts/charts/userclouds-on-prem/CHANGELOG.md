# Changelog

## 1.1.0 - UNRELEASED

- Remove DNS client type from config.
- Fix values file, default image repository for provisionJob

## 1.0.0 - 20-05-2025

NOTE: This version of th chart requires an up to date version of the UserClouds software to work properly.

- Rename console_service_url to console_url in the console config file and code
- Provisioner: Update mounting log DB config (mount from log server base config)
- Move UI assets directory to a subdirectory for each service (plex & console).
- Deduplicate svc_listener config, move to base and have all services use it (port 5000)
- Deduplicate log DB config using a helm template function
- Deduplicate console tenant ID in config
- Eliminate individual service configuration files for on-premises deployments; all configuration is now managed through the Helm chart instead of being embedded in container images
- Consolidate service deployments to use a unified container image, replacing the previous multi-image architecture
- Set default container image repository to `ghcr.io/userclouds/userclouds` and `ghcr.io/userclouds/automatedprovisioner`

## 0.8.0 - 24-04-2025

NOTE: This version of th chart requires an up to date version of the UserClouds software to work properly.

- Don't retry failed jobs, this is to avoid the automatic provisioner to retry the same job multiple times in case of failure.
- Upgrade to redis 7.4.2
- Expose pod IP and Kubernetes node name to the application via environment variables
- Move internal server definition to base on prem config ConfigMap.
- Rename keys on ConfigMap to be more consistent with the rest of the UC Software
- Rename Log DB config key.

## 0.7.0 - 07-02-2025

- Allow configuring OpenTelemetry Collector to send traces to a custom endpoint, via config.openTelemetryEndpoint
- Change internal server port to 5001 (new binaries/container images required), Create a [Prometheus Operator ServiceMonitor CRD](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/running-exporters.md) to collect metrics from services/pods.
- Add the ability to enable Sentry integration.

## 0.6.0 - 03-01-2025

- Configure liveness probes to wait 10 seconds before starting to check the health of the pod
- Upgrade to redis 7.4.1, pull it from AWS ECR, since it is faster when running in AWS and there is no risk of throttling (unlike DockerHub)
- Add the ability to downgrade tenant DB migrations (DO NOT USE IN PRODUCTION)
- Add the ability to add additional annotations to the provision job (automatic provisioner)

## 0.5.1 - 22-10-2024

- Fix DB Proxy NLB annotations indentation so additional annotations are rendered properly

## 0.5.0 - 10-10-2024

- **IMPORTANT** The UC Software now uses the `secretsmanager:TagResource` when creating secrets, so this permissions needs to be added to the IAM role that the UC Software uses.
  Additionally the IAM role needs to have the `secretsmanager:DeleteSecret` permission if the Access Policy Secrets feature is used.
- Simplify redis cache config by making username & password optional (removed them from the configmap used in this chart)
- Fix log server URL in base config file
- Run DB Proxy NLB health checks on on port 1200 instead of 3306, it is configurable from the values file `dbproxy.mysql.healthCheckPort`
- Configure MySQL Ports for DB Proxy NLB in the values file `dbproxy.mysql.ports`

## 0.4.0 - 10-09-2024

- Add logger config to base config file
- Remove the limitation of requiring to install the chart into a specific namespace (`userclouds`) chart can now be installed into any kubernetes namespace.
- Allow configuring resources requests & limits for pods

## 0.3.0 - 05-09-2024

- Update configmap to use the correct pattern for service specific base config files
- Don't use `helm.sh/chart` label is deployment & service selector values since those are immutable.
- Don't hard code namespace for redis in common cache configuration (use `{{ .Release.Namespace }}` instead)

## 0.2.0 - 29-08-2024

- Initial release of the chart
