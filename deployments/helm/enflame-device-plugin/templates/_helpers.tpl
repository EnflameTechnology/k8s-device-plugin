{{/*
Expand the name of the chart.
*/}}
{{- define "enflame-device-plugin.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "enflame-device-plugin.fullname" -}}
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
{{- define "enflame-device-plugin.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "enflame-device-plugin.labels" -}}
helm.sh/chart: {{ include "enflame-device-plugin.chart" . }}
{{ include "enflame-device-plugin.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "enflame-device-plugin.selectorLabels" -}}
app.kubernetes.io/name: {{ include "enflame-device-plugin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Get variant configuration
*/}}
{{- define "enflame-device-plugin.variantConfig" -}}
{{- $variant := .Values.variant -}}
{{- index .Values.variants $variant | toYaml -}}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "enflame-device-plugin.serviceAccountName" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
{{- $variantConfig.serviceAccount.name -}}
{{- end }}

{{/*
Get environment variables with NODE_NAME automatically added
*/}}
{{- define "enflame-device-plugin.environment" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
- name: NODE_NAME
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName
{{- range $key, $value := $variantConfig.env }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- with .Values.overrides.env }}
{{- range $key, $value := . }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Get security context
*/}}
{{- define "enflame-device-plugin.securityContext" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
{{- if eq $variantConfig.securityContext.mode "privileged" }}
{{- $variantConfig.securityContext.container | toYaml }}
{{- else }}
{{- $variantConfig.securityContext.container | toYaml }}
{{- end }}
{{- end }}

{{/*
Get pod security context
*/}}
{{- define "enflame-device-plugin.podSecurityContext" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
{{- if $variantConfig.securityContext.pod }}
{{- $variantConfig.securityContext.pod | toYaml }}
{{- end }}
{{- end }}

{{/*
Get RBAC rules
*/}}
{{- define "enflame-device-plugin.rbacRules" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
{{- $variantConfig.rbac.rules | toYaml }}
{{- end }}

{{/*
Get tolerations
*/}}
{{- define "enflame-device-plugin.tolerations" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
{{- $variantConfig.tolerations | toYaml }}
{{- end }}

{{/*
Get pod annotations
*/}}
{{- define "enflame-device-plugin.annotations" -}}
{{- $variantConfig := include "enflame-device-plugin.variantConfig" . | fromYaml -}}
{{- if $variantConfig.annotations }}
{{- $variantConfig.annotations | toYaml }}
{{- end }}
{{- end }} 