# Marketing landing page (landing/): a React app prerendered to static HTML at build time, served by
# unprivileged nginx. The build context is landing/ (deployments/docker-compose.yml). The fonts are bundled
# from npm and served from this origin, so a visitor's browser never calls a font CDN (GDPR: no IP address
# goes to a third party). Same Node image as node.Dockerfile.
FROM node:26.11.1-trixie-slim AS build
WORKDIR /src
COPY package.json package-lock.json .npmrc ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.31.7-alpine3.24
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist/ /usr/share/nginx/html/
EXPOSE 8080
