package entity

import (
	"encoding/json"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"time"
)

func (c *Content) GetDeployment(domain *valueobject.Domain, hostName string) (*valueobject.Deployment, error) {
	sd, err := c.searchDeployment(domain.QueryString(), hostName)
	if err != nil {
		return nil, err
	}

	if sd == nil {
		item, err := valueobject.NewItemWithNamespace("Deployment")
		if err != nil {
			return nil, err
		}

		sd = &valueobject.Deployment{
			Item:     *item,
			Domain:   domain.QueryString(),
			SiteName: fmt.Sprintf("mdf-%d", time.Now().UnixMilli()),
			HostName: hostName,
			Status:   "pending",
		}

		_, err = c.newContent("Deployment", sd)
		if err != nil {
			return nil, err
		}
	}

	return sd, nil
}

func (c *Content) searchDeployment(domainQueryStr string, hostName string) (*valueobject.Deployment, error) {
	conditions := map[string]string{
		"hash": hash.Fields([]string{domainQueryStr, hostName}),
	}

	// 查询域名信息
	deploys, err := c.termSearch("Deployment", conditions)
	if err != nil {
		return nil, err
	}

	// 如果返回仅一个结果且为 nil，返回 nil
	if len(deploys) == 1 && deploys[0] == nil {
		return nil, nil
	}

	// 遍历查询结果并解析
	for _, data := range deploys {
		var deployment valueobject.Deployment
		if err := json.Unmarshal(data, &deployment); err != nil {
			return nil, err
		}

		// 返回匹配的域名对象
		return &deployment, nil
	}

	// 未找到匹配结果
	return nil, nil
}
