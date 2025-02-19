package entity

import (
	"github.com/mdfriday/hugoverse/internal/domain/resources"
	"github.com/mdfriday/hugoverse/pkg/identity"
	"github.com/mdfriday/hugoverse/pkg/resource/jsconfig"
	"sync"
)

type Common struct {
	Incr identity.Incrementer

	// Assets used after the build is done.
	// This is shared between all sites.
	*PostBuildAssets
}

type PostBuildAssets struct {
	postProcessMu        sync.RWMutex
	PostProcessResources map[string]resources.PostPublishedResource
	JSConfigBuilder      *jsconfig.Builder
}
