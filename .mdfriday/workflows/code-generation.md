{{ define "main" }}

{{ template "code-generation-before.md" . }}

{{ .Content "prompts/ddd.md" }}
{{ .Content "prompts/log.md" }}
{{ .Content "prompts/code-generation.md" }}

{{ template "code-generation-after.md" . }}

{{ end }} 