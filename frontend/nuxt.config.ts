import tailwindcss from '@tailwindcss/vite'

const apiProxy = process.env.NUXT_DEV_API_PROXY || 'http://localhost:8080'

// Pinned: Nuxt ^4.4.8, Vue ^3.5.38, Pinia ^3.0.3, shadcn-nuxt ^2.7.4
export default defineNuxtConfig({
  compatibilityDate: '2025-06-15',
  ssr: false,
  telemetry: false,
  devtools: { enabled: true },
  css: ['~/assets/css/tailwind.css'],
  modules: ['shadcn-nuxt', '@pinia/nuxt'],
  shadcn: {
    prefix: '',
    componentDir: false,
  },
  components: [
    {
      path: '~/components',
      pathPrefix: false,
    },
  ],
  vite: {
    plugins: [tailwindcss()],
    server: {
      proxy: {
        '/api': {
          target: apiProxy,
          changeOrigin: true,
        },
        '/health': {
          target: apiProxy,
          changeOrigin: true,
        },
        '/ready': {
          target: apiProxy,
          changeOrigin: true,
        },
      },
    },
  },
  nitro: {
    devProxy: {
      '/api': {
        target: apiProxy,
        changeOrigin: true,
      },
      '/health': {
        target: apiProxy,
        changeOrigin: true,
      },
      '/ready': {
        target: apiProxy,
        changeOrigin: true,
      },
    },
    routeRules: {
      '/api/**': { proxy: `${apiProxy}/api/**` },
      '/health': { proxy: `${apiProxy}/health` },
      '/ready': { proxy: `${apiProxy}/ready` },
    },
  },
  runtimeConfig: {
    public: {
      // Empty = same-origin relative /api (works with dev proxy or reverse proxy)
      apiUrl: process.env.NUXT_PUBLIC_API_URL || '',
    },
  },
})
