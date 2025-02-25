package entity

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/internal/domain/contenthub"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Hugo struct {
	Services content.Services
	Fs       afero.Fs

	DirService content.DirService

	contentSvc contentSvc

	site *valueobject.Site

	Log loggers.Logger
}

func (h *Hugo) goModFile(dir string) *valueobject.File {
	return &valueobject.File{
		Fs:      h.Fs,
		Path:    path.Join(dir, "go.mod"),
		Content: []byte("module github.com/.mdfriday/temp-build\n\ngo 1.18"),
	}
}

func (h *Hugo) siteConfigFile(site *valueobject.Site, dir string) (*valueobject.File, error) {
	confBytes, err := site.Toml()
	if err != nil {
		h.Log.Errorf("failed to get site config: %v", err)
		return nil, err
	}

	return &valueobject.File{
		Fs:      h.Fs,
		Path:    path.Join(dir, "config.toml"),
		Content: confBytes,
	}, nil
}

func (h *Hugo) previewDir() (string, string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("rand read error: %v", err)
	}

	shortLink := base64.URLEncoding.EncodeToString(b)
	shortLink = strings.TrimRight(shortLink, "=")

	dir := path.Join(h.DirService.PreviewDir(), shortLink)
	if err := h.Fs.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("make preview dir error: %v", err)
	}

	return dir, shortLink, nil
}

func (h *Hugo) previewPath(fullPath string) (string, error) {
	relPath, err := filepath.Rel(h.DirService.DataDir(), fullPath)
	if err != nil {
		return "", fmt.Errorf("get relative path error: %v", err)
	}

	if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}

	return relPath, nil
}

func (h *Hugo) tempDir(prefix string, workingDir string) (string, func(), error) {
	sitesPath := path.Join(workingDir, "sites")
	if err := h.Fs.MkdirAll(sitesPath, 0755); err != nil {
		return "", nil, err
	}

	formattedPrefix := strings.ReplaceAll(prefix, " ", "_") + "_"
	tempDir, err := afero.TempDir(h.Fs, sitesPath, formattedPrefix)
	if err != nil {
		return "", nil, err
	}

	return tempDir, func() { h.Fs.RemoveAll(tempDir) }, nil
}

func (c *Content) LoadHugoProject() error {
	return c.Hugo.LoadProject(c)
}

func (h *Hugo) LoadProject(c contentSvc) error {
	h.contentSvc = c

	if err := h.loadSite(); err != nil {
		return err
	}

	if err := h.loadSiteLanguages(); err != nil {
		return err
	}

	if err := h.loadPosts(); err != nil {
		return err
	}

	return nil
}

func (h *Hugo) loadPosts() error {
	authorQueryStr, err := h.getAuthor()
	if err != nil {
		return err
	}

	codes := h.Services.LanguageKeys()
	for _, code := range codes {
		langIndex, err := h.Services.GetLanguageIndex(code)
		if err != nil {
			return err
		}

		if err := h.Services.WalkPages(langIndex, func(p contenthub.Page) error {
			if p.PageIdentity().PageLanguage() != code {
				return nil
			}

			relName, err := p.PageFile().FileInfo().RelativeFilename()
			if err != nil {
				return err
			}
			h.Log.Printf("Loading post: %s, %s-%s\n", relName, code, p.PageIdentity().PageLanguage())

			i, err := valueobject.NewItemWithNamespace("Post")
			if err != nil {
				return err
			}

			post := &valueobject.Post{
				Item:    *i,
				Title:   p.Title(),
				Author:  authorQueryStr,
				Content: p.PureContent(),
			} // TODO, page assets
			post.Item.Updated = timestamp.TimeMillis(p.PageFile().FileInfo().ModTime())

			id, err := h.contentSvc.newContent("Post", post)
			if err != nil {
				return err
			}

			h.Log.Printf("Loaded post: %+v", post)

			num, err := strconv.Atoi(id)
			if err != nil {
				return err
			}
			post.ID = num

			spi, err := valueobject.NewItemWithNamespace("SitePost")
			if err != nil {
				return err
			}

			relName, err = p.PageFile().FileInfo().RelativeFilename()
			if err != nil {
				return err
			}

			sitePost := &valueobject.SitePost{
				Item: *spi,
				Post: post.QueryString(),
				Site: h.site.QueryString(),
				Path: path.Join(p.PageFile().FileInfo().Root(), relName),
			}
			_, err = h.contentSvc.newContent("SitePost", sitePost)
			if err != nil {
				return err
			}

			h.Log.Printf("Loaded SitePost: %+v", sitePost)

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (h *Hugo) loadSiteLanguages() error {
	codes := h.Services.LanguageKeys()
	for _, code := range codes {
		i, err := valueobject.NewItemWithNamespace("SiteLanguage")
		if err != nil {
			return err
		}

		langQueryStr, err := h.getLanguage(code)
		if err != nil {
			return err
		}
		h.Log.Println("Get language", code, langQueryStr)

		siteLang := &valueobject.SiteLanguage{
			Item:     *i,
			Site:     h.site.QueryString(),
			Language: langQueryStr,
			Default:  code == h.Services.DefaultLanguage(),
			Folder:   h.Services.GetLanguageFolder(code),
		}
		h.Log.Printf("Loadeding SiteLanguage: %+v", *siteLang)
		_, err = h.contentSvc.newContent("SiteLanguage", siteLang)
		if err != nil {
			return err
		}

		h.Log.Printf("Loaded SiteLanguage: %+v", *siteLang)
	}
	return nil
}

func (h *Hugo) loadSite() error {
	i, err := valueobject.NewItemWithNamespace("Site")
	if err != nil {
		return err
	}

	themeQueryStr, err := h.getTheme(h.Services.DefaultTheme())
	if err != nil {
		return err
	}

	site := &valueobject.Site{
		Item:       *i,
		Title:      h.Services.SiteTitle(),
		BaseURL:    h.Services.BaseUrl(),
		WorkingDir: h.Services.WorkingDir(),
		Theme:      themeQueryStr,
		Owner:      "me@sunwei.xyz",
	}
	if h.Services.ConfigParams() != nil {
		site.Params, err = mapToYAML(h.Services.ConfigParams())
		if err != nil {
			return err
		}
	}

	id, err := h.contentSvc.newContent("Site", site)
	if err != nil {
		return err
	}

	h.Log.Printf("Loaded Site: %+v", *site)

	num, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	site.ID = num
	h.site = site

	return nil
}

func (h *Hugo) getTheme(moduleUrl string) (string, error) {
	data, err := h.contentSvc.search("Theme", fmt.Sprintf("module_url:%s", moduleUrl))
	if err != nil {
		return "", err
	}

	if len(data) > 0 {
		firstData := data[0]
		var result valueobject.Theme
		if err := json.Unmarshal(firstData, &result); err != nil {
			return "", err
		}

		return result.QueryString(), nil
	}

	return "", fmt.Errorf("no themes found")
}

func (h *Hugo) getLanguage(code string) (string, error) {
	data, err := h.contentSvc.search("Language", fmt.Sprintf("code:%s", code))
	if err != nil {
		return "", err
	}

	if len(data) > 0 {
		firstData := data[0]
		var result valueobject.Language
		if err := json.Unmarshal(firstData, &result); err != nil {
			return "", err
		}

		return result.QueryString(), nil
	}

	return "", fmt.Errorf("no languages found")
}

func (h *Hugo) getAuthor() (string, error) {
	//TODO get user email from token

	data, err := h.contentSvc.search("Author", fmt.Sprintf("email:%s", "me@sunwei.xyz"))
	if err != nil {
		return "", err
	}

	if len(data) > 0 {
		firstData := data[0]
		var result valueobject.Author
		if err := json.Unmarshal(firstData, &result); err != nil {
			return "", err
		}

		return result.QueryString(), nil
	}

	return "", fmt.Errorf("no authors found")
}

func mapToYAML(data map[string]any) (string, error) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(yamlData), nil
}

func (h *Hugo) syncPostToFilesystem(s *valueobject.Site, p *valueobject.Post, sp *valueobject.SitePost) error {
	absPath := filepath.Join(s.WorkingDir, sp.Path)

	// Check if the file exists
	exists, err := afero.Exists(h.Fs, absPath)
	if err != nil {
		return err
	}

	// If the file does not exist or needs to be updated
	if !exists || h.needsUpdate(absPath, p.Updated) {
		if err := h.writeFileAndUpdateTime(absPath, []byte(p.FullContent()), p.UpdateTime()); err != nil {
			return err
		}
	}

	return nil
}

// needsUpdate checks if the file at absPath needs to be updated based on the provided timestamp.
func (h *Hugo) needsUpdate(absPath string, updated int64) bool {
	fileInfo, err := h.Fs.Stat(absPath)
	if err != nil {
		return true // If we can't stat the file, assume it needs an update
	}
	return timestamp.TimeMillis(fileInfo.ModTime()) < updated
}

// writeFileAndUpdateTime writes the content to the file and updates its modification time.
func (h *Hugo) writeFileAndUpdateTime(absPath string, content []byte, ut time.Time) error {
	if err := h.writeFile(absPath, content); err != nil {
		return err
	}
	if err := h.Fs.Chtimes(absPath, ut, ut); err != nil {
		return err
	}
	return nil
}

func (h *Hugo) writeFile(filename string, data []byte) error {
	err := h.Fs.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		fmt.Println(err)
	}
	err = afero.WriteFile(h.Fs, filename, data, 0666)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}
