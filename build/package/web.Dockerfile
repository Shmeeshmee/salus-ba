# Web app for practitioners and admins (web/): built with Vite and served as static files by unprivileged nginx
# on 8080 (web/nginx.conf.template). The build context is web/. No API proxying here: Caddy routes the API paths of
# web/vite.config.ts to the api service on the same origin (docs/adr/0011-single-server-deployment.md).
# SALUS_DOMAIN (set by the production compose file) names LiveKit's host in the Content-Security-Policy; the
# default is only for local runs. Same Node image as node.Dockerfile.
FROM node:26.13.0-trixie-slim AS build
WORKDIR /src
COPY package.json package-lock.json .npmrc ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

# The base image's Alpine packages can lag behind Alpine's security fixes (a critical brotli fix in 2026-11),
# so the runtime stage upgrades them; the Images workflow rebuilds this stage without the layer cache
# (no-cache-filters: runtime) so each build gets the current fixes.
FROM nginxinc/nginx-unprivileged:1.31.7-alpine3.24@sha256:b5047b9b76845eac124358373c6852377c52e4b05834a47647e955dd91cb8f1e AS runtime
USER root
RUN apk upgrade --no-cache
USER 101
ENV SALUS_DOMAIN=localhost \
    NGINX_ENVSUBST_FILTER=^SALUS_DOMAIN$
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /src/dist/ /usr/share/nginx/html/
EXPOSE 8080
