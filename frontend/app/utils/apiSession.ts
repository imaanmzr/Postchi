export function shouldLogoutOnUnauthorized(refreshSucceeded: boolean): boolean {
  return !refreshSucceeded
}

export async function retryOnUnauthorized(
  execute: () => Promise<Response>,
  refreshAccessToken: () => Promise<boolean>,
): Promise<{ response: Response; refreshSucceeded: boolean }> {
  let refreshSucceeded = false
  let response = await execute()

  if (response.status === 401) {
    refreshSucceeded = await refreshAccessToken()
    if (refreshSucceeded) {
      response = await execute()
    }
  }

  return { response, refreshSucceeded }
}
