package gitrepo

import (
	"net/url"
	"strings"
)

var knownRepoFolderNames = map[string]struct{}{
	"bruno-collection": {},
	"bruno":            {},
	"collections":      {},
	"docs":             {},
	"documentation":    {},
	"openapi":          {},
	"postman":          {},
}

// ParseGitLabTreeRef splits the path segment after /-/tree/ into branch ref and in-repo folder.
func ParseGitLabTreeRef(rest string) (branch, browsePath string) {
	rest = strings.Trim(strings.TrimSpace(rest), "/")
	if rest == "" {
		return "", ""
	}
	segments := strings.Split(rest, "/")
	clean := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			clean = append(clean, segment)
		}
	}
	if len(clean) == 0 {
		return "", ""
	}
	if len(clean) == 1 {
		decoded, _ := url.QueryUnescape(clean[0])
		return decoded, ""
	}

	firstDecoded, _ := url.QueryUnescape(clean[0])
	if strings.Contains(clean[0], "%2F") || strings.Contains(clean[0], "%2f") {
		return firstDecoded, strings.Join(clean[1:], "/")
	}

	last := clean[len(clean)-1]
	if isLikelyRepoFolder(last) {
		decodedBranch, _ := url.QueryUnescape(strings.Join(clean[:len(clean)-1], "/"))
		return decodedBranch, last
	}

	decodedBranch, _ := url.QueryUnescape(clean[0])
	return decodedBranch, strings.Join(clean[1:], "/")
}

func isLikelyRepoFolder(name string) bool {
	if strings.Contains(name, ".") {
		return true
	}
	_, ok := knownRepoFolderNames[strings.ToLower(name)]
	return ok
}

// NormalizePathPrefix removes a duplicate ticket/folder segment copied from the branch name into path prefix.
func NormalizePathPrefix(branch, pathPrefix string) string {
	branch = strings.Trim(branch, "/")
	pathPrefix = strings.Trim(pathPrefix, "/")
	if branch == "" || pathPrefix == "" {
		return pathPrefix
	}
	lastSegment := branch
	if i := strings.LastIndex(branch, "/"); i >= 0 {
		lastSegment = branch[i+1:]
	}
	if lastSegment != "" && strings.HasPrefix(pathPrefix, lastSegment+"/") {
		return strings.TrimPrefix(pathPrefix, lastSegment+"/")
	}
	return pathPrefix
}
