package valueobject

import (
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"strings"
)

// Slug returns a URL friendly string from the title of a post item
func Slug(i content.Identifiable) (string, error) {
	// get the name of the post item
	name := strings.TrimSpace(i.String())

	// filter out non-alphanumeric character or non-whitespace
	slug, err := StringToSlug(name)
	if err != nil {
		return "", err
	}

	return slug, nil
}
