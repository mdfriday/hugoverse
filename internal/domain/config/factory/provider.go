package factory

import (
	"github.com/mdfriday/hugoverse/internal/domain/config/entity"
	"github.com/mdfriday/hugoverse/internal/domain/config/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/mdfriday/hugoverse/pkg/paths"
	"github.com/spf13/afero"
	"os"
	"path"
	"path/filepath"
)

const (
	DefaultThemesDir  = "themes"
	DefaultPublishDir = "public"
)

func LoadConfig() (*entity.Config, error) {
	currentDir, _ := os.Getwd()
	workingDir := filepath.Clean(currentDir)

	l := &ConfigLoader{
		SourceDescriptor: &sourceDescriptor{
			fs:       &afero.OsFs{},
			filename: path.Join(workingDir, "config.toml"),
		},
		Cfg: valueobject.NewDefaultProvider(),
		BaseDirs: valueobject.BaseDirs{
			WorkingDir: workingDir,
			ThemesDir:  paths.AbsPathify(workingDir, DefaultThemesDir),
			PublishDir: paths.AbsPathify(workingDir, DefaultPublishDir),
			CacheDir:   "",
		},
		Logger: loggers.NewDefault(),
	}
	var err error
	l.BaseDirs.CacheDir, err = valueobject.GetCacheDir(l.SourceDescriptor.Fs(), l.BaseDirs.CacheDir)
	if err != nil {
		return nil, err
	}

	defer l.deleteMergeStrategies()
	p, err := l.loadConfigByDefault()
	if err != nil {
		return nil, err
	}

	c := &entity.Config{
		ConfigSourceFs: l.SourceDescriptor.Fs(),
		Provider:       p,

		Root:      entity.Root{},
		Caches:    entity.Caches{},
		Security:  entity.Security{},
		Menu:      entity.Menu{},
		Module:    entity.Module{},
		Service:   entity.Service{},
		Language:  &entity.Language{},
		Imaging:   entity.Imaging{},
		MediaType: entity.MediaType{},
		Sitemap:   entity.Sitemap{},

		Taxonomy: &entity.Taxonomy{},
	}

	if err := l.assembleConfig(c); err != nil {
		return nil, err
	}

	return c, nil
}
