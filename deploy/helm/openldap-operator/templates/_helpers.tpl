{{/*
Expand the name of the chart.
*/}}
{{- define "openldap-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "openldap-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "openldap-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "openldap-operator.labels" -}}
helm.sh/chart: {{ include "openldap-operator.chart" . }}
{{ include "openldap-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: controller
app.kubernetes.io/part-of: openldap-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "openldap-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "openldap-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "openldap-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "openldap-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the cluster role to use
*/}}
{{- define "openldap-operator.clusterRoleName" -}}
{{- printf "%s-manager-role" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create the name of the cluster role binding to use
*/}}
{{- define "openldap-operator.clusterRoleBindingName" -}}
{{- printf "%s-manager-rolebinding" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create the name of the leader election role to use
*/}}
{{- define "openldap-operator.leaderElectionRoleName" -}}
{{- printf "%s-leader-election-role" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create the name of the leader election role binding to use
*/}}
{{- define "openldap-operator.leaderElectionRoleBindingName" -}}
{{- printf "%s-leader-election-rolebinding" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create the docker image name
*/}}
{{- define "openldap-operator.image" -}}
{{- $registry := .Values.global.imageRegistry | default .Values.image.registry -}}
{{- $repository := .Values.image.repository -}}
{{- $tag := .Values.image.tag -}}
{{- if not $tag -}}
{{- $tag = .Chart.AppVersion -}}
{{- end -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry $repository $tag -}}
{{- else -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}
{{- end }}

{{/*
Create image pull secrets
*/}}
{{- define "openldap-operator.imagePullSecrets" -}}
{{- $secrets := .Values.global.imagePullSecrets | default .Values.imagePullSecrets -}}
{{- if $secrets -}}
imagePullSecrets:
{{- range $secrets }}
- name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create namespace for resources
*/}}
{{- define "openldap-operator.namespace" -}}
{{- default .Release.Namespace .Values.namespace }}
{{- end }}

{{/*
Create webhook service name
*/}}
{{- define "openldap-operator.webhookServiceName" -}}
{{- printf "%s-webhook-service" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create metrics service name
*/}}
{{- define "openldap-operator.metricsServiceName" -}}
{{- printf "%s-metrics-service" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create certificate name for webhook
*/}}
{{- define "openldap-operator.certificateName" -}}
{{- printf "%s-serving-cert" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Create issuer name for webhook certificate
*/}}
{{- define "openldap-operator.issuerName" -}}
{{- printf "%s-selfsigned-issuer" (include "openldap-operator.fullname" .) }}
{{- end }}

{{/*
Validate values
*/}}
{{- define "openldap-operator.validateValues" -}}
{{- if and .Values.webhook.enabled (not .Values.webhook.certManager.enabled) (or (not .Values.webhook.certificate.crt) (not .Values.webhook.certificate.key)) -}}
{{- fail "When webhook is enabled and cert-manager is disabled, both webhook.certificate.crt and webhook.certificate.key must be provided" -}}
{{- end -}}
{{- end -}}
