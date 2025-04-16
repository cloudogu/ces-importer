{{/* Chart basics
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec) starting from
Kubernetes 1.4+.
*/}}
{{- define "ces-importer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/* All-in-one labels */}}
{{- define "ces-importer.labels" -}}
app: ces
helm.sh/chart:  {{- printf " %s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "ces-importer.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Selector labels */}}
{{- define "ces-importer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ces-importer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
k8s.cloudogu.com/part-of: ces-migration
{{- end }}

{{- define "ces-importer.sshPrivateKeyPath" -}}
/importerSshPrivateKey
{{- end }}

{{- define "ces-importer.sshUser" -}}
root
{{- end }}