# syntax=docker/dockerfile:1
FROM node:22-alpine AS build
WORKDIR /app
RUN --mount=type=bind,target=/source --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export NODE_EXTRA_CA_CERTS=/run/secrets/proxy_ca; fi; \
    cp -R /source/. /app/ && npm ci && npm run build && rm -rf node_modules /root/.npm
FROM nginx:1.28-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s CMD wget -Y off -q -O /dev/null http://127.0.0.1/ || exit 1
