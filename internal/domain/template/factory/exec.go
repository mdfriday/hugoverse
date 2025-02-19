package factory

import (
	"github.com/mdfriday/hugoverse/internal/domain/template"
	"github.com/mdfriday/hugoverse/internal/domain/template/entity"
	"github.com/mdfriday/hugoverse/internal/domain/template/valueobject"
	"reflect"
)

func New(fs template.Fs, cfs template.CustomizedFunctions) (*entity.Template, error) {
	b := newBuilder().
		withFs(fs).
		withNamespace(newNamespace()).
		withCfs(cfs).
		buildFunctions().
		buildLookup().
		buildParser().
		buildExecutor()

	return b.build()
}

func newLookup(fsv map[string]reflect.Value) *entity.Lookup {
	return &entity.Lookup{
		BaseOf: valueobject.NewBaseOf(),
		Funcsv: fsv,
	}
}

func newNamespace() *entity.Namespace {
	return &entity.Namespace{
		StateMap: &valueobject.StateMap{
			Templates: make(map[string]*valueobject.State),
		},
	}
}
