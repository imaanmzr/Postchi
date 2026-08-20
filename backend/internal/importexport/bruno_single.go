package importexport

import (
	"fmt"
	"strings"

	"github.com/imaanmzr/postchi/backend/internal/importexport/bruno"
	"github.com/imaanmzr/postchi/backend/internal/importexport/model"
	"github.com/imaanmzr/postchi/backend/internal/importexport/validate"
)

func parseBrunoSingleFile(content []byte, filename string) (model.Collection, error) {
	parsed := bruno.Parse(string(content))
	if validate.HasHTTPMethodBlock(parsed) {
		if err := validateBrunoRequest(parsed); err != nil {
			return model.Collection{}, fmt.Errorf("%s: %w", filename, err)
		}
		name := parsed.Name
		if name == "" {
			name = strings.TrimSuffix(filename, ".bru")
		}
		col := model.Collection{
			Name:     name,
			Requests: []model.Request{bruToNorm(bruno.ToRequest(parsed))},
		}
		return col, nil
	}
	if validate.IsCollectionOrFolderMeta(parsed) {
		col := model.Collection{
			Name:      parsed.Name,
			Variables: bruVarsToSpec(bruno.ToVars(parsed.Sections["vars:pre-request"], parsed.Sections["vars:post-response"])),
		}
		if col.Name == "" {
			col.Name = "Imported Bruno"
		}
		return col, nil
	}
	return model.Collection{}, fmt.Errorf("%s: unrecognized Bruno file", filename)
}

func defaultBrunoParseOptions() brunoParseOptions {
	return brunoParseOptions{
		RequireRootMeta:  true,
		ValidateRequests: true,
	}
}
