import { describe, expect, it } from 'vitest'
import {
  applyGitLabBrowseUrlHints,
  detectedGitProvider,
  gitRepoConfigPayload,
  isValidLinkTemplate,
  parseGitLabTreeRef,
  stripGitBrowsePath,
  suggestedLinkTemplate,
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

  it('parses slash branch names and bruno-collection folder', () => {
    const form = {
      repo_url: 'https://git.example.com/acme/repo/-/tree/fix/BO-1287-remove-merchant-domain-check/bruno-collection',
      branch: 'main',
      path_prefix: '',
    }
    applyGitLabBrowseUrlHints(form)
    expect(form.branch).toBe('fix/BO-1287-remove-merchant-domain-check')
    expect(form.path_prefix).toBe('bruno-collection')
  })

  it('normalizes duplicate ticket folder in path prefix on save', () => {
    expect(gitRepoConfigPayload({
      repo_url: 'https://gitlab.com/acme/repo',
      branch: 'fix/BO-1287-remove-merchant-domain-check',
      path_prefix: 'BO-1287-remove-merchant-domain-check/bruno-collection',
    })).toMatchObject({
      branch: 'fix/BO-1287-remove-merchant-domain-check',
      path_prefix: 'bruno-collection',
    })
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

  it('parses two-segment slash branch names', () => {
    expect(parseGitLabTreeRef('feature/unmerged-docs')).toEqual({
      branch: 'feature/unmerged-docs',
      path: '',
    })
  })

  it('strips browse paths from saved repo URLs', () => {
    expect(stripGitBrowsePath('https://gitlab.com/acme/repo/-/tree/feature/docs')).toBe('https://gitlab.com/acme/repo')
    expect(gitRepoConfigPayload({
      repo_url: 'https://gitlab.com/acme/repo/-/tree/feature/unmerged/docs',
      branch: 'feature/unmerged',
      path_prefix: 'docs',
      link_template: 'docs/{request_slug}.md',
    })).toEqual({
      repo_url: 'https://gitlab.com/acme/repo',
      branch: 'feature/unmerged',
      path_prefix: 'docs',
      link_template: 'docs/{request_slug}.md',
    })
  })

  it('validates link templates', () => {
    expect(isValidLinkTemplate('docs/{request_slug}.md')).toBe(true)
    expect(isValidLinkTemplate('docs/{operation_id}.md')).toBe(true)
    expect(isValidLinkTemplate('docs')).toBe(false)
    expect(suggestedLinkTemplate('docs')).toBe('{path_prefix}/{request_slug}.md')
  })

  it('rejects static link templates in payload', () => {
    expect(() => gitRepoConfigPayload({
      repo_url: 'https://github.com/acme/repo',
      branch: 'feature/unmerged',
      path_prefix: 'docs',
      link_template: 'docs',
    })).toThrow(/link template/i)
  })
})
