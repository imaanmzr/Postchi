package gitsync

import (
	"context"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/imaanmzr/postchi/backend/internal/importexport/detect"
	"github.com/imaanmzr/postchi/backend/internal/shared/gitrepo"
)

const (
	MaxFiles       = 2_000
	MaxFileBytes   = 2 << 20
	MaxTotalBytes  = 32 << 20
	MaxConcurrency = 8
)

type SourceFile struct {
	Path    string
	Content []byte
}

type DiscoveredRoot struct {
	Format   detect.Format
	RootPath string
	Files    []SourceFile
	FilePath string
	Content  []byte
}

func FetchRepository(ctx context.Context, client *gitrepo.Client) ([]SourceFile, error) {
	paths, err := client.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0, len(paths))
	for _, filePath := range paths {
		relative := relativeRepositoryPath(filePath, client.PathPrefix)
		if relative == "" {
			continue
		}
		parts := strings.Split(relative, "/")
		if len(parts) > 0 && containsPathSegment(parts[:len(parts)-1], "environments") {
			continue
		}
		candidates = append(candidates, filePath)
		if len(candidates) > MaxFiles {
			return nil, &gitrepo.Error{
				Kind:    gitrepo.ErrorLimit,
				Message: "repository contains more than 2,000 importable files; use a narrower path prefix",
			}
		}
	}
	if len(candidates) == 0 {
		return nil, &gitrepo.Error{Kind: gitrepo.ErrorNotFound, Message: "repository contains no importable collection files"}
	}

	type fetchResult struct {
		file SourceFile
		err  error
	}
	results := make(chan fetchResult, len(candidates))
	semaphore := make(chan struct{}, MaxConcurrency)
	var group sync.WaitGroup
	for _, filePath := range candidates {
		group.Add(1)
		go func(repoPath string) {
			defer group.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results <- fetchResult{err: ctx.Err()}
				return
			}
			content, fetchErr := client.FetchFile(ctx, repoPath)
			results <- fetchResult{
				file: SourceFile{Path: relativeRepositoryPath(repoPath, client.PathPrefix), Content: content},
				err:  fetchErr,
			}
		}(filePath)
	}
	go func() {
		group.Wait()
		close(results)
	}()

	files := make([]SourceFile, 0, len(candidates))
	totalBytes := 0
	for result := range results {
		if result.err != nil {
			return nil, result.err
		}
		totalBytes += len(result.file.Content)
		if totalBytes > MaxTotalBytes {
			return nil, &gitrepo.Error{
				Kind:    gitrepo.ErrorLimit,
				Message: "repository files exceed the 32 MiB total import limit",
			}
		}
		files = append(files, result.file)
	}
	return files, nil
}

func Discover(files []SourceFile) []DiscoveredRoot {
	claimed := map[string]bool{}
	var roots []DiscoveredRoot

	brunoRoots := findBrunoRoots(files)
	for _, rootPath := range brunoRoots {
		subtree := collectBrunoSubtree(files, rootPath, brunoRoots, claimed)
		if len(subtree) == 0 {
			continue
		}
		roots = append(roots, DiscoveredRoot{
			Format:   detect.FormatBruno,
			RootPath: rootPath,
			Files:    subtree,
		})
	}

	for _, file := range files {
		if claimed[file.Path] {
			continue
		}
		lower := strings.ToLower(file.Path)
		if strings.HasSuffix(lower, ".bru") {
			continue
		}
		format := detect.DetectFormat(file.Path, file.Content)
		switch format {
		case detect.FormatPostman, detect.FormatOpenCollection, detect.FormatOpenAPI:
			roots = append(roots, DiscoveredRoot{
				Format:   format,
				FilePath: file.Path,
				Content:  file.Content,
			})
		}
	}
	return roots
}

func findBrunoRoots(files []SourceFile) []string {
	var roots []string
	seen := map[string]bool{}
	for _, file := range files {
		base := strings.ToLower(filepath.Base(file.Path))
		if base != "collection.bru" {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(file.Path))
		if dir == "." {
			dir = ""
		}
		dir = strings.Trim(dir, "/")
		if seen[dir] {
			continue
		}
		seen[dir] = true
		roots = append(roots, dir)
	}
	return roots
}

func collectBrunoSubtree(files []SourceFile, rootPath string, allRoots []string, claimed map[string]bool) []SourceFile {
	var out []SourceFile
	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Path), ".bru") {
			continue
		}
		if !fileBelongsToBrunoRoot(file.Path, rootPath, allRoots) {
			continue
		}
		parts := strings.Split(file.Path, "/")
		if containsPathSegment(parts[:len(parts)-1], "environments") {
			continue
		}
		claimed[file.Path] = true
		out = append(out, file)
	}
	return out
}

func fileBelongsToBrunoRoot(path, root string, allRoots []string) bool {
	if root != "" {
		if path != root+"/collection.bru" && !strings.HasPrefix(path, root+"/") {
			return false
		}
		for _, other := range allRoots {
			if other == root || other == "" {
				continue
			}
			if strings.HasPrefix(other, root+"/") &&
				(path == other+"/collection.bru" || strings.HasPrefix(path, other+"/")) {
				return false
			}
		}
		return true
	}
	if !strings.Contains(path, "/") {
		return true
	}
	for _, other := range allRoots {
		if other == "" {
			continue
		}
		if path == other+"/collection.bru" || strings.HasPrefix(path, other+"/") {
			return false
		}
	}
	return true
}

func RelativeRepositoryPath(filePath, prefix string) string {
	return relativeRepositoryPath(filePath, prefix)
}

func relativeRepositoryPath(filePath, prefix string) string {
	cleanPath := strings.Trim(path.Clean("/"+filePath), "/")
	cleanPrefix := strings.Trim(path.Clean("/"+prefix), "/")
	if cleanPrefix == "" || cleanPrefix == "." {
		return cleanPath
	}
	if cleanPath == cleanPrefix {
		return ""
	}
	return strings.TrimPrefix(cleanPath, cleanPrefix+"/")
}

func containsPathSegment(parts []string, target string) bool {
	for _, part := range parts {
		if strings.EqualFold(part, target) {
			return true
		}
	}
	return false
}
