# Marketing landing page (landing/): one static HTML file served by unprivileged nginx. The build context is
# landing/ (deployments/docker-compose.yml). Inter and Inter Tight are fetched from npm at build time and
# served from the same origin, so a visitor's browser never calls a font CDN (GDPR: no IP sent to a third party).
FROM node:24.22.0-trixie-slim AS fonts
WORKDIR /work
RUN mkdir /out \
 && npm pack --silent @fontsource-variable/inter@5 @fontsource-variable/inter-tight@5 \
 && for pkg in *.tgz; do \
      tar -xzf "$pkg" \
      && cp package/files/*-latin-wght-*.woff2 package/files/*-latin-ext-wght-*.woff2 /out/ \
      && rm -rf package; \
    done

FROM nginxinc/nginx-unprivileged:1.31.7-alpine3.24
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY index.html /usr/share/nginx/html/index.html
COPY --from=fonts /out/ /usr/share/nginx/html/fonts/
EXPOSE 8080
