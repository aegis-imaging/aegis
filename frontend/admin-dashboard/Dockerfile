FROM node:20-alpine AS build
WORKDIR /app

COPY package*.json ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# VITE_WEASIS_BASE_URL is baked into the static build at compile time.
# Set this to the Weasis Cloud Run service URL for production builds.
ARG VITE_WEASIS_BASE_URL
ENV VITE_WEASIS_BASE_URL=$VITE_WEASIS_BASE_URL

COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime
# gettext provides envsubst, used to hydrate API_URL into nginx config at startup.
RUN apk add --no-cache gettext

# Default: GCP API URL. Override with API_URL env var for other clouds (e.g. AWS ECS).
ENV API_URL=https://api.aegisimaging.ai
# Default: GCP MCP server URL. Override with MCP_SERVER_URL env var for other clouds (e.g. AWS ECS).
ENV MCP_SERVER_URL=https://aegis-mcp-server-uk5cvzf5nq-uc.a.run.app

COPY nginx.conf /etc/nginx/conf.d/nginx.conf.template
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
# Substitute ${API_URL} in the nginx config template, then start nginx.
CMD ["/bin/sh", "-c", "envsubst '${API_URL} ${MCP_SERVER_URL}' < /etc/nginx/conf.d/nginx.conf.template > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"]
