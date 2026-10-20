# Node toolchain for web/: npm scripts, the Vite dev server and the TypeScript client generator. The npm
# cache lives in a named volume at /cache; world-writable so the container can run as the host user.
FROM node:24.22.0-trixie-slim
RUN mkdir -p /cache/npm /cache/home && chmod -R 0777 /cache
ENV npm_config_cache=/cache/npm \
    npm_config_update_notifier=false \
    npm_config_fund=false \
    HOME=/cache/home
WORKDIR /src/web
