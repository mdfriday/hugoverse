package handler

import (
	"encoding/json"
	"errors"
	"github.com/mdfriday/hugoverse/pkg/herrors"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"net/http"
	"strings"
)

func (s *Handler) handlerErrorWithLog(fields *loggers.LogFields, res http.ResponseWriter, req *http.Request, err error) {
	s.log.Error().WithFields(fields).WithError(err).Logf("req: %s", req.URL.String())

	s.handlerError(res, req, err)
}

func (s *Handler) handlerError(res http.ResponseWriter, req *http.Request, err error) {
	var fe herrors.FileError
	if errors.As(err, &fe) {
		res.WriteHeader(http.StatusBadRequest)

		jsonBytes, err := json.Marshal(extractErrorMessage(fe.Error()))
		if err != nil {
			s.log.Errorf("Error marshalling token when handling error: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		j, err := s.res.FmtJSON(jsonBytes)
		if err != nil {
			s.log.Errorf("Error formatting JSON when handling error: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		s.res.Json(res, j)

		return
	}

	res.WriteHeader(http.StatusInternalServerError)
	_, err = res.Write([]byte(err.Error()))
	if err != nil {
		s.log.Errorf("Error writing response: %v", err)
	}

	return
}

func extractErrorMessage(errMsg string) string {
	// 查找 "content" 的位置
	if idx := strings.Index(errMsg, "content"); idx != -1 {
		return errMsg[idx:]
	}
	return errMsg
}

// Error is the message and http status code to return
type Error struct {
	Message string
	Code    int
}

// BadRequest is a convenience function for returning a bad request error
func BadRequest(message string) *Error {
	return &Error{
		Message: message,
		Code:    http.StatusBadRequest,
	}
}

// InternalServerError is a convenience function for returning an internal server error
func InternalServerError() *Error {
	return &Error{
		Message: "Something went wrong",
		Code:    http.StatusInternalServerError,
	}
}
