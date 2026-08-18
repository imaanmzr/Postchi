package model

import (
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/shared/domain"
)

type Collection struct {
	Name             string
	Description      string
	SortOrder        int
	Variables        domain.VariablesSpec
	Headers          []request.KVPair
	Auth             request.AuthSpec
	PreRequestScript string
	TestScript       string
	Children         []Collection
	Requests         []Request
}

type Request struct {
	Name             string
	Method           string
	URL              string
	Description      string
	Headers          []request.KVPair
	Params           []request.KVPair
	PathVars         []request.KVPair
	Body             request.BodySpec
	Auth             request.AuthSpec
	Settings         request.Settings
	PreRequestScript string
	TestScript       string
	SortOrder        int
}
