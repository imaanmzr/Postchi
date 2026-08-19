import { safeAuthRedirect } from '~/utils/authRedirect'

const publicPaths = ['/login', '/register']

export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) return

  const auth = useAuthStore()
  const isPublic = publicPaths.includes(to.path)
    || to.path.startsWith('/invite/')
    || to.path.startsWith('/share/')

  if (isPublic) {
    if (to.path === '/login' || to.path === '/register') {
      const valid = await auth.ensureSession()
      if (valid) return navigateTo(safeAuthRedirect(to.query.redirect))
    }
    return
  }

  const valid = await auth.ensureSession()
  if (!valid) {
    return navigateTo({
      path: '/login',
      query: { redirect: to.fullPath },
    })
  }
})
