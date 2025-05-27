# Userclouds helm chart for on prem deployments

## Introduction

The Userclouds Helm chart is designed for deploying Userclouds services in on-premises environments using Kubernetes, specifically tailored for Amazon EKS. This chart simplifies the deployment process by providing pre-configured templates and settings, ensuring that all necessary components are deployed consistently and efficiently. With built-in support for PostgreSQL, Google OAuth, and AWS Secrets Manager, the chart facilitates secure and scalable deployments, allowing organizations to leverage the full power of Kubernetes while maintaining control over their infrastructure.

## Prerequisites

* Helm 3.15 or higher
* Kubernetes 1.30+ on AWS EKS

## Set up AWS Environment and EKS Cluster

* Create a namespace in the EKS cluster to install the userclouds software into.
All the resources (secrets) mentioned in the following steps should be created in the namespace that was created in this stepThis can be done by running the following command:

```shell
kubectl create namespace userclouds
```

Recommended when using AWS ALB Ingress Controller: Add the [pod readiness gate](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.9/deploy/pod_readiness_gate/) label to the namespace to ensure that the pods are ready before the service is marked as ready.

```shell
kubectl label namespace userclouds elbv2.k8s.aws/pod-readiness-gate-inject=enabled
```

* Provision an AWS RDS Aurora Postgres instance, it is recommended to use the latest version of Postgres 14 as newer version  (15) currently have incompatibilities with the Userclouds software.
Once provisioned, create a k8s secret with the postgres password.
Create secret with the postgres password

```shell
kubectl create secret generic postgresql-creds -n userclouds --from-literal=POSTGRES_PASSWORD=yourpassword
```

* Create secret to use with the userclouds api

```shell
kubectl create secret generic userclouds-api-client-secret -n userclouds --from-literal=client_secret=<client secret>
```

* Configure a Google OAuth Client ID and Secret for the console to use for social logins.

    1. Go to <https://console.developers.google.com/apis/credentials>.
    2. Click Create Credentials, then click OAuth Client ID in the drop-down menu
    3. Enter the following:
            Application Type: Web Application
            Name: UserClouds Console
            Authorized JavaScript Origins: <https://console.uc.mycompany.com>
            Authorized Redirect URLs: <https://console.tenant.uc.mycompany.com/social/callback>
            Replace <uc.mycompany.com> with the the domain you choose for your UserClouds Console.
    4. Click Create
    5. Copy the Client ID and Client Secret from the 'OAuth Client' modal and store them info a kubernetes secret.

```shell
kubectl create secret generic userclouds-google-auth -n userclouds --from-literal=client_id=<client id> --from-literal=client_secret=<client secret>
```

* Create IAM Role, policy and attach to a service account

The UserClouds software uses AWS Secrets Manager to store secrets, it is hard coded to put all secrets under the `userclouds/onprem` path.
so the following policy needs to be attached to the IAM role:

```json
{
    "Statement": [
        {
            "Action": [
                "secretsmanager:DeleteSecret",  // Optional, only needed if using the Access Policy Secrets feature
                "secretsmanager:UpdateSecret",
                "secretsmanager:GetSecretValue",
                "secretsmanager:TagResource",
                "secretsmanager:CreateSecret"
            ],
            "Effect": "Allow",
            "Resource": "arn:aws:secretsmanager:*:<>AWS Account ID>:secret:userclouds/onprem/*"
        }
    ],
    "Version": "2012-10-17"
}
```

Configure the IAM role to use [AWS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) for the cluster. note that the trust policy for the IAM role needs to reference both the namespace and service account in the k8s cluster.
the service account name can be configured via the helm values file under `serviceAccount.name` (defaults to `userclouds-onprem`) and the namespace is the namespace created in the first step.
Make note of the IAM role ARN.

* [AWS ALB Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/latest/) - This will create & manage AWS ALB for the console and webapp based on ingress objects.

## Image Repository

The UserClouds services deployment uses a unified container image approach:

* All services (authz, userstore, console, etc.) use a single container image, specified by `image.repository` and `image.tag`
* Each service deployment passes specific command-line arguments to determine which service to run
* The provisioning job uses a separate image repository (`provisionJob.image.repository`) but inherits the same tag and pull policy from the main image configuration

## Configure the helm chart

* Set the values you want to change in the values.yaml file

| Parameter                                              | Description                                                         | Default                                                              |
|--------------------------------------------------------|---------------------------------------------------------------------|----------------------------------------------------------------------|
| `image.repository`                                     | Image repository                                                    | `ghcr.io/userclouds/userclouds`                                                                   |
| `image.tag`                                            | Image tag                                                           | ``                                                                   |
| `image.pullPolicy`                                     | Image pull policy                                                   | `IfNotPresent`                                                       |
| `provisionJob.additionalAnnotations`                   | Annotations to add to the automated provisioner job object          | `{}`                                                                 |
| `provisionJob.image.repository`                        | Provision job image repository                                      | `ghcr.io/userclouds/automatedprovisioner`                                                                   |
| `serviceAccount.name`                                  | Service account name                                                | `userclouds-on-prem`                                                 |
| `serviceAccount.iamRoleARN`                            | IAM role ARN                                                        | ``                                                                   |
| `serviceMonitor.enabled`                               | Create a ServiceMonitor CRD (if the CRD is available)               | true                                                                 |
| `config.openTelemetryEndpoint`                         | Endpoint for open telemetry collector in the form of 'host:port'    | ``                                                                   |
| `config.sentry.enabled`                                | Enable sentry integration                                           | `false`                                                              |
| `config.sentry.dsn`                                    | Sentry DSN                                                          | ``                                                                   |
| `config.sentry.sample_rate`                            | Sentry tracing sample rate (zero disables tracing)                  | 0                                                                    |
| `config.replicas`                                      | number of replicas for services pods                                | 3                                                                    |
| `config.downMigrateTenantDBVersion`                    | [DO NOT USE] Automated provisioner will to downgrade DB schema      | ``                                                                   |
| `config.companyName`                                   | Company name                                                        | ``                                                                   |
| `config.customerDomain`                                | Customer domain                                                     | ``                                                                   |
| `config.adminUserEmail`                                | Admin user email                                                    | ``                                                                   |
| `config.db.user`                                       | Username for postgres database                                      | ``                                                                   |
| `config.db.host`                                       | Host for postgres database                                          | ``                                                                   |
| `config.db.port`                                       | Port for postgres database                                          | 5432                                                                 |
| `config.skipEnsureAWSSecretsAccess`                    | Skips checking AWS Secrets Manager access                           | `false`                                                              |
| `userclouds.nodeSelector`                              | Node selector for userclouds pods                                   | `{}`                                                                 |
| `redis.nodeSelector`                                   | Node selector for the redis pod                                     | `{}`                                                                 |
| `console.ingress.enabled`                              | Enable ingress for the console                                      | `false`                                                              |
| `console.ingress.scheme`                               | Scheme for the console ingress                                      | `internet-facing`                                                    |
| `console.ingress.additionalAnnotations`                | Additional annotations for the console ingres                       | `{}`                                                                 |
| `webapp.ingress.enabled`                               | Enable ingress for the webapp                                       | `false`                                                              |
| `webapp.ingress.scheme`                                | Scheme for the webapp ingress                                       | `internet-facing`                                                    |
| `webapp.ingress.additionalAnnotations`                 | Additional annotations for the webapp ingress                       | `{}`                                                                 |
| `dbproxy.mysql.healthCheckPort`                        | Health check port for DB Proxy NLB for the                          | `1200`                                                               |
| `dbproxy.mysql.ports`                                  | Ports to expose for accepting MySQL DB connections                  | `3306, 3307, 3308, 3309, 3310, 3311, 3312, 3313, 3314, 3315`         |
| `dbproxy.mysql.ingress.enabled`                        | Enable ingress (NLB) for the DB Proxy                               | `false`                                                              |
| `dbproxy.mysql.ingress.scheme`                         | Scheme for the DB Proxy NLB                                         | `internal`                                                           |
| `dbproxy.mysql.ingress.additionalAnnotations`          | Additional annotations for DB Proxy NLB                             | `{}`                                                                 |

* Note that the `config.skipEnsureAWSSecretsAccess` is only used in the provisioning job. Once the system is up and running, this flag should be flipped to `true`.
