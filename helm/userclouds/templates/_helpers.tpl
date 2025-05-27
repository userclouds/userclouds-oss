
{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "userclouds.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Userclouds binary container image
*/}}
{{- define "userclouds.image" -}}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end }}


{{/*
Common labels
*/}}
{{- define "userclouds.labels" -}}
helm.sh/chart: {{ include "userclouds.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: userclouds
{{- end }}

{{- define "userclouds.nodeSelector" -}}
provisioner: karpenter
kubernetes.io/arch: amd64
{{- end }}

{{- define "userclouds.workerClientConfig" -}}
worker_client:
    type: sqs
    url: {{ .Values.config.common.worker_queue_url }}
{{- end }}

{{- define "userclouds.openSearchConfig" -}}
{{- with  .Values.config.common.opensearch }}
opensearch:
  url: {{ .url }}
  region: {{ .region }}
  max_results: 1000
{{- end }}
{{- end }}


{{/*
To prevent errors (http 502 from load balancer and request timeouts during restarts (deploy)
we add this lifecycle hook which sleeps for a few seconds before the pod is terminated, but after it has been removed from the service endpoint list (and thus deregistered from the ALB).
more context: https://github.com/foriequal0/pod-graceful-drain
  https://github.com/kubernetes-sigs/aws-load-balancer-controller/pull/1775#issuecomment-773912940
  https://github.com/kubernetes-sigs/aws-load-balancer-controller/issues/1719 https://github.com/kubernetes-sigs/aws-load-balancer-controller/issues/1065
*/}}
{{- define "userclouds.lifecycleDrain" -}}
lifecycle:
  preStop:
    exec:
      command: ["sleep", "{{.Values.preStopSleepSeconds}}"]
{{- end }}


{{/*
Optional prevent pods from being scheduled on t3 machines
*/}}
{{- define "userclouds.maybeNoT3Nodes" -}}
{{- if (default false .dont_allow_t3) }}
nodeAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    nodeSelectorTerms:
      - matchExpressions:
        - key: karpenter.k8s.aws/instance-family
          operator: NotIn
          values:
          - t3
{{- end }}
{{- end }}


{{/*
Common env variables, usage in infra/kubernetes/helpers.go
*/}}
{{- define "userclouds.envVars" -}}
- name: IS_KUBERNETES
  value: "true"
- name: "POD_NAME"
  valueFrom:
    fieldRef:
        fieldPath: metadata.name
- name: K8S_NODE_NAME
  valueFrom:
    fieldRef:
        fieldPath: spec.nodeName
- name: K8S_POD_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: K8S_POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: UC_CONFIG_DIR
  value: /userclouds/configmaps
{{- end }}

{{- define "userclouds.envConfigMap" -}}
envFrom:
  - configMapRef:
      # See terraform/modules/userclouds/serving/eks-k8s-resources/main.tf
      name: userclouds-env-variables
{{- end }}

# // Must match terraform/modules/userclouds/serving/eks-iam-roles/main.tf
{{- define "authz.serviceAccount" -}}
{{- printf "authz"  }}
{{- end }}

{{- define "checkattribute.serviceAccount" -}}
{{- printf "checkattribute"  }}
{{- end }}

{{- define "userstore.serviceAccount" -}}
{{- printf "userstore"  }}
{{- end }}

{{- define "plex.serviceAccount" -}}
{{- printf "plex"  }}
{{- end }}

{{- define "console.serviceAccount" -}}
{{- printf "console"  }}
{{- end }}

{{- define "logserver.serviceAccount" -}}
{{- printf "logserver"  }}
{{- end }}
{{- define "worker.serviceAccount" -}}
{{- printf "worker"  }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "authz.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: authz
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "checkattribute.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: checkattribute
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "userstore.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: userstore
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "dbproxy.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: dbproxy
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "plex.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: plex
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "console.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: console
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "logserver.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: logserver
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "worker.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/component: worker
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
