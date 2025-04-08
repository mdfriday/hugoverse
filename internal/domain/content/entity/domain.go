package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

func (c *Content) ApplyDomain(siteId string, domain string) (*valueobject.Domain, bool, error) {
	site, err := c.getContent("Site", siteId)
	if err != nil {
		c.Log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("site id: %s, domain : %s", siteId, domain)
		return nil, false, err
	}

	if site, ok := site.(*valueobject.Site); ok {
		var slug string
		if site.SubDomain == "" {
			slug, err = valueobject.Slug(site) // Title
			if err != nil {
				c.Log.Error().
					WithFields(loggers.GetGlobalFields()).
					WithError(err).
					Logf("Applying empty domain slug : %v, site : %s", err, site.String())
				return nil, false, err
			}
		} else {
			slug, err = valueobject.StringToSlug(site.SubDomain)
			if err != nil {
				c.Log.Error().
					WithFields(loggers.GetGlobalFields()).
					WithError(err).
					Logf("Applying with sub domain : %v, slug : %s, sub domain : %s ", err, slug, site.SubDomain)
				return nil, false, err
			}
		}

		sd, err := c.searchDomain(domain, slug)
		if err != nil {
			c.Log.Error().
				WithFields(loggers.GetGlobalFields()).
				WithError(err).
				Logf("Applying domain search domain error : %v, domain : %s, slug : %s", err, domain, slug)
			return nil, false, err
		}

		if sd != nil && sd.Owner != site.Owner {
			return nil, true, errors.New(fmt.Sprintf("domain %s already exists", sd.String()))
		}

		if sd == nil {
			item, err := valueobject.NewItemWithNamespace("Domain")
			if err != nil {
				return nil, false, err
			}

			sd = &valueobject.Domain{
				Item:  *item,
				Root:  domain,
				Sub:   slug,
				Owner: site.Owner,
			}

			_, err = c.newContent("Domain", sd)
			if err != nil {
				return nil, false, err
			}
		}

		return sd, false, nil
	}

	return nil, false, errors.New("only site could be deployed with domain")
}

func (c *Content) searchDomain(root string, sub string) (*valueobject.Domain, error) {
	// 构建精确匹配的查询条件
	conditions := map[string]string{
		"hash": hash.Fields([]string{sub, root}),
	}

	// 查询域名信息
	domains, err := c.termSearch("Domain", conditions)
	if err != nil {
		return nil, err
	}

	// 如果返回仅一个结果且为 nil，返回 nil
	if len(domains) == 1 && domains[0] == nil {
		return nil, nil
	}

	// 遍历查询结果并解析
	for _, data := range domains {
		if string(data) == "[]" {
			c.Log.Println("Empty array received for domain search")
			c.Log.Debug().
				WithFields(loggers.GetGlobalFields()).
				Logf("Empty array received for domain search - continue")
			continue
		}
		var domain valueobject.Domain
		if err := json.Unmarshal(data, &domain); err != nil {
			c.Log.Error().
				WithFields(loggers.GetGlobalFields()).
				WithError(err).
				Logf("search domain json.Unmarshal error: %s", string(data))
			continue
		}

		// 返回匹配的域名对象
		return &domain, nil
	}

	// 未找到匹配结果
	return nil, nil
}
