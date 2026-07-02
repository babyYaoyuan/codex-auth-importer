package importer

import (
	"encoding/json"
	"net/http"
)

const (
	contentTypeHTML = "text/html; charset=utf-8"
	contentTypeJSON = "application/json; charset=utf-8"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

func htmlResponse(statusCode int, body []byte) managementResponse {
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type": []string{contentTypeHTML},
		},
		Body: body,
	}
}

func jsonResponse(statusCode int, payload any) managementResponse {
	raw, errMarshal := json.MarshalIndent(payload, "", "  ")
	if errMarshal != nil {
		raw = []byte(`{"error":"failed to marshal response"}`)
	}
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type": []string{contentTypeJSON},
		},
		Body: raw,
	}
}

func okEnvelope(v any) ([]byte, error) {
	raw, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func ErrorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	return raw
}
