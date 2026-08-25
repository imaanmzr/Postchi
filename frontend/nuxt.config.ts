import tailwindcss from '@tailwindcss/vite'

const apiProxy = process.env.NUXT_DEV_API_PROXY || 'http://localhost:8080'

// Pinned: Nuxt ^4.4.8, Vue ^3.5.38, Pinia ^3.0.3, shadcn-nuxt ^2.7.4
export default defineNuxtConfig({
  compatibilityDate: '2025-06-15',
  ssr: false,
  telemetry: false,
  devtools: { enabled: true },
  css: ['~/assets/css/tailwind.css'],
  app: {
    head: {
      title: 'Postchi',
      titleTemplate: '%s · Postchi',
      meta: [
        { name: 'description', content: 'Self-hosted API collaboration platform.' },
        { name: 'theme-color', content: '#1a1b26' },
        { property: 'og:title', content: 'Postchi' },
        { property: 'og:description', content: 'Self-hosted API collaboration platform.' },
        { property: 'og:type', content: 'website' },
        { property: 'og:image', content: '/brand/og-image.png' },
        { name: 'twitter:card', content: 'summary_large_image' },
      ],
      link: [
        { rel: 'icon', href: '/brand/favicon.ico', sizes: '48x48' },
        { rel: 'icon', type: 'image/svg+xml', href: '/brand/icon.svg' },
        { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/brand/favicon-32.png' },
        { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/brand/favicon-16.png' },
        { rel: 'apple-touch-icon', sizes: '180x180', href: '/brand/apple-touch-icon.png' },
      ],
    },
  },
  modules: ['shadcn-nuxt', '@pinia/nuxt', '@vite-pwa/nuxt'],
  pwa: {
    registerType: 'prompt',
    manifest: {
      name: 'Postchi',
      short_name: 'Postchi',
      description: 'Self-hosted API collaboration platform.',
      id: '/',
      start_url: '/',
      scope: '/',
      display: 'standalone',
      orientation: 'any',
      theme_color: '#1a1b26',
      background_color: '#1a1b26',
      categories: ['developer', 'productivity'],
      icons: [
        {
          src: '/brand/icon-192.png',
          sizes: '192x192',
          type: 'image/png',
          purpose: 'any',
        },
        {
          src: '/brand/icon-512.png',
          sizes: '512x512',
          type: 'image/png',
          purpose: 'any',
        },
        {
          src: '/brand/icon-maskable-512.png',
          sizes: '512x512',
          type: 'image/png',
          purpose: 'maskable',
        },
        {
          src: '/brand/icon.svg',
          sizes: 'any',
          type: 'image/svg+xml',
          purpose: 'any',
        },
      ],
    },
    workbox: {
      navigateFallback: '/',
      navigateFallbackDenylist: [/^\/api\//, /^\/health$/, /^\/ready$/],
      cleanupOutdatedCaches: true,
      clientsClaim: true,
    },
    client: {
      installPrompt: 'postchi:pwa-install-dismissed',
      periodicSyncForUpdates: 3600,
    },
    devOptions: {
      enabled: true,
      type: 'module',
    },
  },
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