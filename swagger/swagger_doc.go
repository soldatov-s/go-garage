package swagger

import (
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
)

type Doc struct {
	swagger *openapi3.T
}

func NewDoc(swagger *openapi3.T) *Doc {
	return &Doc{
		swagger: swagger,
	}
}

func (s *Doc) ReadDoc() string {
	return string(s.JSON())
}

func (s *Doc) JSON() (jsonData []byte) {
	jsonData, err := json.MarshalIndent(s.swagger, "", "  ")
	if err != nil {
		return
	}
	return
}
