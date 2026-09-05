package gitrepo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Client) gitLabTreeNotFoundError(ctx context.Context, status int, body []byte) error {
	if status != http.StatusNotFound {
		return responseErrorBody("GitLab", status, body)
	}
	branchExists, err := c.gitLabBranchExists(ctx)
	if err != nil {
		return responseErrorBody("GitLab", status, body)
	}
	if !branchExists {
		return &Error{
			Kind:    ErrorNotFound,
			Message: fmt.Sprintf("GitLab branch %q was not found on the remote. Use Load branches to pick the correct feature branch, or paste a GitLab /-/tree/branch/folder URL in the repository field.", c.Branch),
			Status:  status,
		}
	}
	if strings.Trim(c.PathPrefix, "/") != "" {
		return &Error{
			Kind:    ErrorNotFound,
			Message: fmt.Sprintf(
				"GitLab path %q was not found on branch %q. Path prefix is the folder inside the repo on that branch (for example bruno-collection), not the branch or ticket name.",
				displayPath(c.PathPrefix),
				c.Branch,
			),
			Status: status,
		}
	}
	return &Error{
		Kind:    ErrorNotFound,
		Message: fmt.Sprintf("GitLab branch %q exists but contains no accessible files at the repository root.", c.Branch),
		Status:  status,
	}
}

func (c *Client) gitLabBranchExists(ctx context.Context) (bool, error) {
	apiRoot := strings.TrimSuffix(c.APIBaseURL, "/") + "/api/v4"
	branchURL := fmt.Sprintf("%s/projects/%s/repository/branches/%s",
		apiRoot, encodeGitLabPath(c.RepoPath), encodeGitLabPath(c.Branch))
	resp, err := c.doRequest(ctx, http.MethodGet, branchURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, responseError("GitLab", resp)
	}
}

func (c *Client) gitLabFileNotFoundError(ctx context.Context, filePath string, status int, body []byte) error {
	if status != http.StatusNotFound {
		return responseErrorBody("GitLab", status, body)
	}
	branchExists, err := c.gitLabBranchExists(ctx)
	if err != nil {
		return responseErrorBody("GitLab", status, body)
	}
	if !branchExists {
		return &Error{
			Kind:    ErrorNotFound,
			Message: fmt.Sprintf("GitLab branch %q was not found while fetching %q", c.Branch, filePath),
			Status:  status,
		}
	}
	return &Error{
		Kind:    ErrorNotFound,
		Message: fmt.Sprintf("GitLab file %q was not found on branch %q", filePath, c.Branch),
		Status:  status,
	}
}

func gitHubTreeNotFoundError(c *Client, status int, body []byte) error {
	if status != http.StatusNotFound {
		return responseErrorBody("GitHub", status, body)
	}
	if strings.Trim(c.PathPrefix, "/") != "" {
		return &Error{
			Kind:    ErrorNotFound,
			Message: fmt.Sprintf(
				"GitHub path %q was not found on branch %q. Path prefix is the folder inside the repo on that branch, not the branch name.",
				displayPath(c.PathPrefix),
				c.Branch,
			),
			Status: status,
		}
	}
	return &Error{
		Kind:    ErrorNotFound,
		Message: fmt.Sprintf("GitHub branch %q was not found or is empty.", c.Branch),
		Status:  status,
	}
}
