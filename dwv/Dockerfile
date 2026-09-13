# ── Stage 1: Build DWV viewer app ────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# ── Stage 2: Serve with nginx ─────────────────────────────────────────────────
FROM nginx:1.31-alpine

# gettext provides envsubst for runtime API_URL substitution in nginx config
RUN apk add --no-cache gettext

# Serve DWV static build
COPY --from=builder /app/dist /usr/share/nginx/html

# nginx config template — ${API_URL} is substituted at container startup
COPY nginx.conf.template /nginx.conf.template

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
