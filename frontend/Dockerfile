FROM node:24.19.0-alpine AS builder
WORKDIR /app
ARG NUXT_PUBLIC_API_URL=
ENV NUXT_PUBLIC_API_URL=${NUXT_PUBLIC_API_URL}
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM node:24.19.0-alpine
WORKDIR /app
COPY --from=builder /app/.output ./.output
EXPOSE 3000
ENV NUXT_HOST=0.0.0.0
ENV NUXT_PORT=3000
CMD ["node", ".output/server/index.mjs"]
