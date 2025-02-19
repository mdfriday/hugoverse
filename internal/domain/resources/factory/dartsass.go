package factory

import (
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/resources"
	"github.com/mdfriday/hugoverse/internal/domain/resources/entity"
	"github.com/mdfriday/hugoverse/internal/domain/resources/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hexec"
)

func newDartSass(exec *hexec.Exec, fs resources.Fs) (*entity.SassClient, error) {
	valueobject.SetDartSassBinaryName()
	if valueobject.DartSassBinaryName == "" {
		return &entity.SassClient{BinaryFound: false}, fmt.Errorf("no Dart Sass binary found in $PATH")
	}

	if err := exec.Sec().CheckAllowedExec(valueobject.DartSassBinaryName); err != nil {
		return &entity.SassClient{BinaryFound: true, AllowedExec: false}, err
	}

	sc := &entity.SassClient{
		BinaryFound: true,
		AllowedExec: true,
		FsService:   fs,
	}
	if err := sc.Open(); err != nil {
		return sc, err
	}

	return sc, nil
}
