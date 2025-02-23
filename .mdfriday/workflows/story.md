{{ define "main" }}

{{ template "story-before.md" . }}
{{ .Content "prompts/story.md" }}
{{ template "story-after.md" . }}

{{ end }} 