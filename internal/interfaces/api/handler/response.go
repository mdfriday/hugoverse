package handler

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type ResponseErrView interface {
	Error400() ([]byte, error)
	Error500() ([]byte, error)
}

type Response struct {
	errView ResponseErrView
}

func NewResponse(errView ResponseErrView) *Response {
	return &Response{
		errView: errView,
	}
}

// Json should be used any time you want to communicate
// data back to a foreign client
func (s *Response) Json(res http.ResponseWriter, data []byte) {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Vary", "Accept-Encoding")

	_, err := res.Write(data)
	if err != nil {
		log.Println("Error writing to response in sendData")
	}
}

func (s *Response) FmtJSON(data ...json.RawMessage) ([]byte, error) {
	var resp map[string][]json.RawMessage

	if len(data) == 0 {
		// Explicitly handle empty data case to ensure "data": [] in response
		resp = map[string][]json.RawMessage{
			"data": []json.RawMessage{},
		}
	} else if len(data) == 1 {
		resp = map[string][]json.RawMessage{
			"data": data,
		}
	} else {
		var msg []json.RawMessage
		for _, d := range data {
			msg = append(msg, d)
		}

		resp = map[string][]json.RawMessage{
			"data": msg,
		}
	}

	var buf = &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(resp)
	if err != nil {
		log.Println("Failed to encode data to JSON:", err)
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Response) err400(res http.ResponseWriter) error {
	res.WriteHeader(http.StatusBadRequest)
	errView, err := s.errView.Error400()
	if err != nil {
		return err
	}

	_, err = res.Write(errView)
	if err != nil {
		return err
	}
	return nil
}

func (s *Response) err500(res http.ResponseWriter) error {
	res.WriteHeader(http.StatusInternalServerError)
	errView, err := s.errView.Error500()
	if err != nil {
		return err
	}

	_, err = res.Write(errView)
	if err != nil {
		return err
	}
	return nil
}
