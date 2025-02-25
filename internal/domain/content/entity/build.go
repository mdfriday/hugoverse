package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/fs/static"
	"github.com/mdfriday/hugoverse/pkg/herrors"
	"github.com/spf13/afero"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

func (c *Content) BuildTarget(contentType, id, status string) (string, error) {
	content, err := c.getContent(contentType, id)
	if err != nil {
		return "", err
	}

	if site, ok := content.(*valueobject.Site); ok {
		dir, _, err := c.Hugo.tempDir(site.Title, c.Repo.UserDataDir())
		//defer cleanFunc()
		c.Log.Debugf("temp dir: %s", dir)

		if err != nil {
			c.Log.Errorf("failed to get temp dir: %v", err)
			return "", err
		}

		writer := newWriter(c.Log)
		go writer.startDumpFiles()

		writer.files <- &valueobject.File{
			Fs:      c.Hugo.Fs,
			Path:    path.Join(dir, "go.mod"),
			Content: []byte("module github.com/.mdfriday/temp-build\n\ngo 1.18"),
		}

		confFile, err := c.Hugo.siteConfigFile(site, dir)
		if err != nil {
			c.Log.Errorf("failed to get site config file: %v", err)
			writer.close()
			return "", err
		}
		writer.files <- confFile

		if err := c.writeSitePosts(site.ID, dir, writer.files); err != nil {
			c.Log.Errorf("failed to write site posts: %v", err)
			writer.close()
			return "", err
		}

		if err := c.writeSiteResource(site.ID, dir); err != nil {
			c.Log.Errorf("failed to write site resources: %v", err)
			writer.close()
			return "", err
		}

		writer.close()

		err = <-writer.errs
		if err != nil {
			return "", fmt.Errorf("failed to render pages: %w", herrors.ImproveIfNilPointer(err))
		}

		if site.WorkingDir != "" {
			info, err := os.Stat(site.WorkingDir)
			if err == nil && info.IsDir() {
				info, err := os.Stat(path.Join(site.WorkingDir, "resources"))
				if err == nil && info.IsDir() {
					_, err := os.Stat(path.Join(dir, "resources"))
					if os.IsNotExist(err) {
						from := afero.NewBasePathFs(c.Hugo.Fs, path.Join(site.WorkingDir, "resources"))
						to := afero.NewBasePathFs(c.Hugo.Fs, path.Join(dir, "resources"))
						err := static.Copy([]afero.Fs{from}, to)
						if err != nil {
							c.Log.Errorf("failed to reuse resources: %v", err)
						}
					}
				}
			}
		}

		site.WorkingDir = dir
		if err := c.UpdateContentObject(site); err != nil {
			c.Log.Errorf("Error updating site preview working dir: %v", err)
		}

		return dir, nil
	}

	return "", errors.New("only site could be built")
}

func (c *Content) writeSitePosts(siteId int, dir string, writerFiles chan *valueobject.File) error {
	q := fmt.Sprintf(`site%d`, siteId)

	sitePosts, err := c.Repo.ContentByPrefix(GetNamespace("SitePost", ""), q)
	if err != nil {
		return err
	}

	c.Log.Debugf("sitePosts len: %d", len(sitePosts))

	for _, data := range sitePosts {
		var sp valueobject.SitePost
		if err := json.Unmarshal(data, &sp); err != nil {
			return err
		}

		post, err := c.getPost(sp.Post)
		if err != nil {
			return err
		}

		confBytes, err := post.Markdown()
		if err != nil {
			c.Log.Errorf("failed to get site config: %v", err)
			return err
		}

		writerFiles <- &valueobject.File{
			Fs:      c.Hugo.Fs,
			Path:    path.Join(dir, sp.Path),
			Content: confBytes,
		}

		if len(post.Assets) > 0 {
			go c.copyFiles(dir, getParentPath(sp.Path), post.Assets)
		}
	}

	return nil
}

func (c *Content) writeSiteResource(siteId int, dir string) error {
	q := fmt.Sprintf(`site%d`, siteId)

	siteResources, err := c.Repo.ContentByPrefix(GetNamespace("SiteResource", ""), q)
	if err != nil {
		return err
	}

	for _, data := range siteResources {
		var sr valueobject.SiteResource
		if err := json.Unmarshal(data, &sr); err != nil {
			return err
		}

		res, err := c.getResource(sr.Resource)
		if err != nil {
			return err
		}

		if res.Asset != "" {
			_, p, err := parseURL(res.Asset)
			if err != nil {
				c.Log.Printf("parse url %s failed: %v\n", res.Asset, err)
				return err
			}

			src := path.Join(c.Hugo.DirService.UploadDir(), p)
			dst := path.Join(dir, sr.Path)

			if err := c.Hugo.Fs.MkdirAll(path.Join(dir, getParentPath(sr.Path)), 0755); err != nil {
				c.Log.Printf("mkdir %s failed: %v\n", path.Join(dir, getParentPath(sr.Path)), err)
				continue
			}

			if err := c.copyFile(src, dst); err != nil {
				return err
			}
		}
	}

	return nil
}

func getParentPath(fullPath string) string {
	// 获取父目录路径
	dirPath := filepath.Dir(fullPath)

	// 去除开头的斜杠
	return path.Clean(dirPath)[1:]
}

func (c *Content) copyFiles(dir string, parentPath string, files []string) {
	var wg sync.WaitGroup

	for _, filePath := range files {
		wg.Add(1)

		filename, p, err := parseURL(filePath)
		if err != nil {
			c.Log.Printf("parse url %s failed: %v\n", filePath, err)
			wg.Done()
			continue
		}

		src := path.Join(c.Hugo.DirService.UploadDir(), p)
		dst := path.Join(dir, parentPath, filename)

		if err := c.Hugo.Fs.MkdirAll(path.Join(dir, parentPath), 0755); err != nil {
			c.Log.Printf("mkdir %s failed: %v\n", path.Join(dir, parentPath), err)
			wg.Done()
			continue
		}

		go func(src, dst string) {
			defer wg.Done()

			if err := c.copyFile(src, dst); err != nil {
				return
			}
		}(src, dst)
	}

	wg.Wait()
}

func parseURL(url string) (string, string, error) {
	apiIndex := strings.Index(url, "/api/uploads/")
	if apiIndex == -1 {
		return "", "", fmt.Errorf("URL not contain /api/uploads/, path: %s", url)
	}

	apiPath := url[apiIndex+len("/api/uploads/"):]

	fileName := path.Base(apiPath)

	return fileName, apiPath, nil
}

func (c *Content) copyFile(src, dst string) error {
	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot open source file: %v", err)
	}
	defer sourceFile.Close()

	// 创建目标文件
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("cannot create destination file: %v", err)
	}
	defer destFile.Close()

	// 复制文件内容
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("copy file failed: %v", err)
	}

	return nil
}

func (c *Content) getPost(rawURL string) (*valueobject.Post, error) {
	id, err := c.getIDByURL(rawURL)
	if err != nil {
		return nil, err
	}

	p, err := c.getContent("Post", id)
	if err != nil {
		return nil, err
	}

	post, ok := p.(*valueobject.Post)
	if !ok {
		return nil, errors.New("invalid post")
	}

	return post, nil
}

func (c *Content) getResource(rawURL string) (*valueobject.Resource, error) {
	id, err := c.getIDByURL(rawURL)
	if err != nil {
		return nil, err
	}

	p, err := c.getContent("Resource", id)
	if err != nil {
		return nil, err
	}

	post, ok := p.(*valueobject.Resource)
	if !ok {
		return nil, errors.New("invalid post")
	}

	return post, nil
}

func (c *Content) getIDByURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url failed: %v", err)
	}

	id := parsedURL.Query().Get("id")
	if id == "" {
		return "", fmt.Errorf("cannot get id from url: %s", rawURL)
	}

	return id, nil
}
