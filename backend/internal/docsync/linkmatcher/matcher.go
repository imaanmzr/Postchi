package linkmatcher

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
)

type Doc struct {
	ID                 string
	Slug               string
	Title              string
	SourcePath         string
	ContentMD          string
	LinkedOperationIDs []string
}

type Request struct {
	ID                string
	Name              string
	Method            string
	URL               string
	SourceOperationID string
	CollectionName    string
}

type Candidate struct {
	DocID      string
	RequestID  string
	Reason     string
	Confidence string
	Evidence   map[string]string
}

var methodPathRe = regexp.MustCompile(`(?i)\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(/[^\s\])"'` + "`" + `]+)`)

func Analyze(docs []Doc, requests []Request, skip func(docID, requestID string) bool) []Candidate {
	var out []Candidate
	for _, doc := range docs {
		for _, req := range requests {
			if skip(doc.ID, req.ID) {
				continue
			}
			if c, ok := matchContentMethodPath(doc, req); ok {
				out = append(out, c)
				continue
			}
			if c, ok := matchPathAlignment(doc, req); ok {
				out = append(out, c)
				continue
			}
			if c, ok := matchTitleSimilarity(doc, req); ok {
				out = append(out, c)
				continue
			}
			if c, ok := matchFolderAlignment(doc, req); ok {
				out = append(out, c)
			}
		}
	}
	return out
}

func matchContentMethodPath(doc Doc, req Request) (Candidate, bool) {
	canonical := operationid.CanonicalFromMethodURL(req.Method, req.URL)
	if canonical == "" {
		return Candidate{}, false
	}
	matches := methodPathRe.FindAllStringSubmatch(doc.ContentMD, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		found := operationid.CanonicalFromMethodURL(m[1], m[2])
		if found == canonical {
			return Candidate{
				DocID: doc.ID, RequestID: req.ID,
				Reason: "content_method_path", Confidence: "high",
				Evidence: map[string]string{"match": m[0], "canonical": canonical},
			}, true
		}
	}
	return Candidate{}, false
}

func matchPathAlignment(doc Doc, req Request) (Candidate, bool) {
	docSeg := lastPathSegment(doc.SourcePath, doc.Slug)
	reqSeg := lastPathSegment(operationid.CanonicalFromMethodURL(req.Method, req.URL), req.Name)
	if docSeg == "" || reqSeg == "" {
		return Candidate{}, false
	}
	docSeg = normalizeToken(docSeg)
	reqSeg = normalizeToken(reqSeg)
	if docSeg == reqSeg || strings.Contains(docSeg, reqSeg) || strings.Contains(reqSeg, docSeg) {
		return Candidate{
			DocID: doc.ID, RequestID: req.ID,
			Reason: "path_alignment", Confidence: "high",
			Evidence: map[string]string{"doc_segment": docSeg, "request_segment": reqSeg},
		}, true
	}
	return Candidate{}, false
}

func matchTitleSimilarity(doc Doc, req Request) (Candidate, bool) {
	docTokens := tokenSet(doc.Title + " " + doc.Slug)
	reqTokens := tokenSet(req.Name)
	if len(docTokens) == 0 || len(reqTokens) == 0 {
		return Candidate{}, false
	}
	overlap := 0
	for t := range reqTokens {
		if docTokens[t] {
			overlap++
		}
	}
	score := float64(overlap) / float64(len(reqTokens))
	if score >= 0.6 && overlap >= 2 {
		return Candidate{
			DocID: doc.ID, RequestID: req.ID,
			Reason: "title_similarity", Confidence: "medium",
			Evidence: map[string]string{"score": formatScore(score)},
		}, true
	}
	return Candidate{}, false
}

func matchFolderAlignment(doc Doc, req Request) (Candidate, bool) {
	docFolder := folderPath(doc.SourcePath)
	reqFolder := normalizeToken(req.CollectionName)
	if docFolder == "" || reqFolder == "" {
		return Candidate{}, false
	}
	docFolderNorm := normalizeToken(strings.ReplaceAll(docFolder, "/", " "))
	if strings.Contains(docFolderNorm, reqFolder) || strings.Contains(reqFolder, docFolderNorm) {
		return Candidate{
			DocID: doc.ID, RequestID: req.ID,
			Reason: "folder_alignment", Confidence: "medium",
			Evidence: map[string]string{"doc_folder": docFolder, "collection": req.CollectionName},
		}, true
	}
	return Candidate{}, false
}

func lastPathSegment(paths ...string) string {
	for _, p := range paths {
		p = strings.Trim(p, "/")
		if p == "" {
			continue
		}
		p = strings.ReplaceAll(p, "-", "/")
		parts := strings.Split(p, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			seg := strings.TrimSpace(parts[i])
			if seg != "" {
				return seg
			}
		}
	}
	return ""
}

func folderPath(sourcePath string) string {
	sourcePath = strings.Trim(sourcePath, "/")
	if sourcePath == "" {
		return ""
	}
	if i := strings.LastIndex(sourcePath, "/"); i >= 0 {
		return sourcePath[:i]
	}
	return ""
}

func tokenSet(s string) map[string]bool {
	s = normalizeToken(s)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make(map[string]bool)
	for _, p := range parts {
		if len(p) >= 2 {
			out[p] = true
		}
	}
	return out
}

func normalizeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Join(strings.Fields(s), " ")
}

func formatScore(v float64) string {
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), `"`)
}

// EvidenceJSON marshals evidence for DB storage.
func EvidenceJSON(ev map[string]string) []byte {
	if ev == nil {
		ev = map[string]string{}
	}
	b, _ := json.Marshal(ev)
	return b
}
