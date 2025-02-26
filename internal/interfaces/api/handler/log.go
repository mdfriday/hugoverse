package handler

import (
	"github.com/bep/logg"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

func (s *Handler) newLogFields(operation string) *loggers.LogFields {
	return loggers.NewLogFieldsWithCommon(operation, s.auth.Session).
		AddFields(
			logg.Field{Name: "user_id", Value: s.auth.UserId},
		)
}
