import { describe, expect, it } from 'vitest'
import {
  applyGitLabBrowseUrlHints,
  detectedGitProvider,
  gitRepoConfigPayload,
} from './gitRepoForm'

describe('gitRepoForm', () => {
  it('detects GitHub and GitLab providers', () => {
    expect(detectedGitProvider('https://github.com/acme/repo')).toBe('GitHub')
    expect(detectedGitProvider('https://gitlab.com/acme/repo/-/tree/main/bruno')).toBe('GitLab')
  })

  it('fills branch and path from GitLab tree URLs', () => {
    const form = {
      repo_url: 'https://gitlab.com/acme/repo/-/tree/develop/collections/api',
      branch: 'main',
      path_prefix: '',
    }
    applyGitLabBrowseUrlHints(form)
    expect(form.branch).toBe('develop')
    expect(form.path_prefix).toBe('collections/api')
  })

  it('builds import payload with defaults', () => {
    expect(gitRepoConfigPayload({
      repo_url: ' https://github.com/acme/repo ',
      branch: '',
      path_prefix: ' bruno ',
    })).toEqual({
      repo_url: 'https://github.com/acme/repo',
      branch: 'main',
      path_prefix: 'bruno',
    })
  })
})
