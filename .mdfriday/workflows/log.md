{{ define "main" }}

{{ .Content "prompts/1.global/log.md" }}
{{ .Content "prompts/2.program/hv-golang-backend-log.md" }}
{{ .Content "prompts/2.program/hv-golang-backend-log-format.md" }}
{{ .Content "prompts/2.program/hv-golang-backend-log-performance.md" }}
{{ .Content "prompts/3.solution/golang-logger.md" }}

{{ end }} 