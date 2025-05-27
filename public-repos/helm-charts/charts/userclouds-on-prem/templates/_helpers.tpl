
{{- define "userclouds.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

# https://helm.sh/docs/chart_best_practices/labels/#standard-labels

{{- define "userclouds.selectorLabels" -}}
app.kubernetes.io/name: userclouds
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "userclouds.labels" -}}
helm.sh/chart:  {{ include "userclouds.chart" . }}
{{ include "userclouds.selectorLabels" . }}
{{- end }}

{{/*
Userclouds binary container image
*/}}
{{- define "userclouds.image" -}}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end }}

{{/*
Userclouds provision job image
*/}}
{{- define "userclouds.automated_provisioner_image" -}}
{{- printf "%s:%s" .Values.provisionJob.image.repository .Values.image.tag }}
{{- end }}

{{- define "userclouds.envVars" -}}
- name: "POD_NAME"
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: K8S_POD_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: K8S_NODE_NAME
  valueFrom:
    fieldRef:
        fieldPath: spec.nodeName
- name: K8S_POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
# config/base_onprem.yaml
- name: PG_PASSWORD
  valueFrom:
    secretKeyRef:
      name: postgresql-creds
      key: password
- name: UC_API_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: userclouds-api-client-secret
      key: client_secret
- name: UC_CONFIG_DIR
  value: /userclouds/configmaps
- name: UC_UNIVERSE
  value: onprem
- name: UC_REGION
  value: customerlocal
- name: UC_ON_PREM_CUSTOMER_DOMAIN
  value: ".{{ .Values.config.customerDomain }}"
{{- end }}


{{- define "userclouds.envVars.googleAuth" -}}
- name: GOOGLE_CLIENT_ID
  valueFrom:
    secretKeyRef:
      name: userclouds-google-auth
      key: client_id
- name: GOOGLE_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: userclouds-google-auth
      key: client_secret
{{- end }}

{{- define "userclouds.logdb_config" -}}
log_db:
  user: {{ .Values.config.db.user }}
  password: env://PG_PASSWORD
  dbname: status_00000000000000000000000000000000
  host: {{ .Values.config.db.host }}
  dbdriver: postgres
  dbproduct: aws-rds-postgres
  port: {{ .Values.config.db.port }}
{{- end }}
