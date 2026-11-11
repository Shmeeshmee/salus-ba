# Marketing landing page (landing/): a React app prerendered to static HTML at build time, served by
# unprivileged nginx. The build context is landing/ (deployments/docker-compose.yml). The fonts are bundled
# from npm and served from this origin, so a visitor's browser never calls a font CDN (GDPR: no IP address
# goes to a third party). Same Node image as node.Dockerfile.
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
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist/ /usr/share/nginx/html/
EXPOSE 8080
