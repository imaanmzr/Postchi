package domain

import "encoding/json"

type PreRequestVar struct {
	Enabled     bool   `json:"enabled"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Secret      bool   `json:"secret"`
}

type PostResponseVar struct {
	Enabled     bool   `json:"enabled"`
	Name        string `json:"name"`
	Expr        string `json:"expr"`
	Description string `json:"description"`
}

type VariablesSpec struct {
	PreRequest   []PreRequestVar   `json:"pre_request"`
	PostResponse []PostResponseVar `json:"post_response"`
}

func EmptyVariablesSpec() VariablesSpec {
	return VariablesSpec{
		PreRequest:   []PreRequestVar{},
		PostResponse: []PostResponseVar{},
	}
}

func ParseVariablesSpec(data []byte) VariablesSpec {
	if len(data) == 0 {
		return EmptyVariablesSpec()
	}
	var spec VariablesSpec
	if err := json.Unmarshal(data, &spec); err == nil && (spec.PreRequest != nil || spec.PostResponse != nil) {
		if spec.PreRequest == nil {
			spec.PreRequest = []PreRequestVar{}
		}
		if spec.PostResponse == nil {
			spec.PostResponse = []PostResponseVar{}
		}
		return spec
	}
	// Legacy flat map
	var flat map[string]string
	if json.Unmarshal(data, &flat) == nil {
		spec := EmptyVariablesSpec()
		for k, v := range flat {
			spec.PreRequest = append(spec.PreRequest, PreRequestVar{
				Enabled: true, Name: k, Value: v, Type: "string",
			})
		}
		return spec
	}
	return EmptyVariablesSpec()
}

func (v VariablesSpec) ToMap() map[string]string {
	out := map[string]string{}
	for _, row := range v.PreRequest {
		if row.Enabled && row.Name != "" {
			out[row.Name] = row.Value
		}
	}
	return out
}

func MergeVariablesSpecMaps(specs ...VariablesSpec) map[string]string {
	out := map[string]string{}
	for _, spec := range specs {
		for k, v := range spec.ToMap() {
			out[k] = v
		}
	}
	return out
}
