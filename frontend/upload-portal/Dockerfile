FROM node:20-alpine AS build
WORKDIR /app

COPY package*.json ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# VITE_INVITE_GATE_ENABLED controls the invite code gate at build time.
# Set to "true" in production to require an invite code before showing the portal.
# Default is "false" (open access for local dev).
ARG VITE_INVITE_GATE_ENABLED
ENV VITE_INVITE_GATE_ENABLED=$VITE_INVITE_GATE_ENABLED

COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime
# gettext provides envsubst, used to hydrate API_URL into nginx config at startup.
RUN apk add --no-cache gettext

# Default: GCP API URL. Override with API_URL env var for other clouds (e.g. AWS ECS).
ENV API_URL=https://api.aegisimaging.ai

COPY nginx.conf /etc/nginx/conf.d/nginx.conf.template
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
# Substitute ${API_URL} in the nginx config template, then start nginx.
CMD ["/bin/sh", "-c", "envsubst '${API_URL} ${NGINX_RESOLVER_DIRECTIVE}' < /etc/nginx/conf.d/nginx.conf.template > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"]
