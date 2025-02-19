package entity

import (
	"context"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/contenthub"
	"github.com/mdfriday/hugoverse/pkg/maps"
	"path/filepath"
	"strings"
)

func (s *Site) Description() string {
	return "A Hugoverse Site built with love. "
}

func (s *Site) Params() maps.Params {
	cp := s.ConfigSvc.ConfigParams()
	ps := s.Reserve.Contact()

	maps.MergeParams(cp, ps)

	return cp
}

func (s *Site) Home() *Page {
	return s.home
}

func (s *Site) Sections() []*Page {
	pgs := s.home.Page.Sections(s.CurrentLanguageIndex())

	return s.sitePages(pgs)
}

func (s *Site) IsMultiLingual() bool {
	return s.Language.isMultipleLanguage()
}

func (s *Site) GetPage(ref ...string) (*Page, error) {
	if len(ref) > 1 {
		// This was allowed in Hugo <= 0.44, but we cannot support this with the
		// new API. This should be the most unusual case.
		return nil, fmt.Errorf(`too many arguments to .Site.GetPage: %v. Use lookups on the form {{ .Site.GetPage "/posts/mypage-md" }}`, ref)
	}

	key := ref[0]
	key = filepath.ToSlash(key)
	if !strings.HasPrefix(key, "/") {
		key = "/" + key
	}

	p, err := s.ContentSvc.GetPageFromPath(s.CurrentLanguageIndex(), key)
	if err != nil {
		return nil, err
	}

	if p == nil {
		return nil, nil
	}

	return s.sitePage(p)
}

func (s *Site) Pages() Pages {
	cps := s.ContentSvc.GlobalPages(s.CurrentLanguageIndex())

	return s.sitePages(cps)
}

func (s *Site) RegularPages() Pages {
	cps := s.ContentSvc.GlobalRegularPages()

	return s.sitePages(cps)
}

func (s *Site) pageOutput(p contenthub.Page) (contenthub.PageOutput, error) {
	pos, err := p.PageOutputs()
	if err != nil {
		return nil, err
	}
	if len(pos) != 1 {
		return nil, fmt.Errorf("expected 1 page output, got %d", len(pos))
	}
	po := pos[0] // TODO, check for multiple outputs

	return po, nil
}

func (s *Site) sitePages(ps contenthub.Pages) []*Page {
	var pages []*Page
	for _, cp := range ps {
		np, err := s.sitePage(cp)
		if err != nil {
			continue
		}

		pages = append(pages, np)
	}

	return pages
}

func (s *Site) sitePage(p contenthub.Page) (*Page, error) {
	po, err := s.pageOutput(p)
	if err != nil {
		return nil, err
	}

	sp := &Page{
		resSvc:    s.ResourcesSvc,
		tmplSvc:   s.Template,
		langSvc:   s.LanguageSvc,
		publisher: s.Publisher,
		git:       s.GitSvc,

		Page:       p,
		PageOutput: po,
		Site:       s,
	}

	sources, err := s.ContentSvc.GetPageSources(sp.Page)
	if err != nil {
		return nil, err
	}

	if err := sp.processResources(sources); err != nil {
		return nil, err
	}

	return sp, nil
}

func (s *Site) siteWeightedPage(p contenthub.OrdinalWeightPage) (*WeightedPage, error) {
	sp, err := s.sitePage(p.Page())
	if err != nil {
		return nil, err
	}

	return &WeightedPage{sp, p}, nil
}

func (s *Site) Translate(ctx context.Context, translationID string, templateData any) string {
	return s.TranslationSvc.Translate(ctx, s.Language.currentLanguage, translationID, templateData)
}
