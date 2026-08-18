import { formatResponseBody } from './formatResponseBody'

export interface RequestBodyShape {
  mode: string
  raw: string
  raw_lang: string
  graphql?: { query: string; variables: string }
}

function isJsonRequestBody(body: RequestBodyShape): boolean {
  return body.mode === 'json' || (body.mode === 'raw' && body.raw_lang === 'json')
}

/** Pretty-print JSON request bodies the same way as responses. */
export function formatRequestBody<T extends RequestBodyShape>(body: T): T {
  let next = body

  if (isJsonRequestBody(body) && body.raw) {
    const formatted = formatResponseBody(body.raw)
    if (formatted !== body.raw) {
      next = { ...next, raw: formatted }
    }
  }

  if (body.graphql?.variables) {
    const formatted = formatResponseBody(body.graphql.variables)
    if (formatted !== body.graphql.variables) {
      next = {
        ...next,
        graphql: { ...body.graphql, variables: formatted },
      }
    }
  }

  return next
}
