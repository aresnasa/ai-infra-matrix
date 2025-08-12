{{/*
Expand the name of the chart.
*/}}
{{- define "ai-infra-matrix.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ai-infra-matrix.fullname" -}}
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
{{- define "ai-infra-matrix.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ai-infra-matrix.labels" -}}
helm.sh/chart: {{ include "ai-infra-matrix.chart" . }}
{{ include "ai-infra-matrix.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ai-infra-matrix.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ai-infra-matrix.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component labels
*/}}
{{- define "ai-infra-matrix.componentLabels" -}}
{{- $component := . -}}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "ai-infra-matrix.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ai-infra-matrix.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the user namespace
*/}}
{{- define "ai-infra-matrix.userNamespace" -}}
{{- default "ai-infra-users" .Values.userNamespace }}
{{- end }}

{{/*
Image repository with registry
*/}}
{{- define "ai-infra-matrix.imageRepository" -}}
{{- $repository := .repository -}}
{{- $values := .values -}}
{{- if $values.global.imageRegistry }}
{{- printf "%s/%s" $values.global.imageRegistry $repository }}
{{- else if $values.image.registry }}
{{- printf "%s/%s" $values.image.registry $repository }}
{{- else }}
{{- $repository }}
{{- end }}
{{- end }}

{{/*
Database connection string
*/}}
{{- define "ai-infra-matrix.databaseUrl" -}}
{{- if .Values.postgresql.enabled }}
{{- $host := printf "%s-postgresql" (include "ai-infra-matrix.fullname" .) }}
{{- $port := "5432" }}
{{- $database := .Values.postgresql.auth.database }}
{{- $username := .Values.postgresql.auth.username }}
{{- printf "postgresql://%s:$(POSTGRES_PASSWORD)@%s:%s/%s" $username $host $port $database }}
{{- else }}
{{- printf "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)" }}
{{- end }}
{{- end }}

{{/*
Redis connection details
*/}}
{{- define "ai-infra-matrix.redisHost" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis-master" (include "ai-infra-matrix.fullname" .) }}
{{- else }}
{{- "redis-service" }}
{{- end }}
{{- end }}

{{/*
Backend service URL
*/}}
{{- define "ai-infra-matrix.backendUrl" -}}
{{- printf "http://%s-backend:%d" (include "ai-infra-matrix.fullname" .) (.Values.backend.service.port | int) }}
{{- end }}
