FROM node:20-alpine AS build
WORKDIR /app

COPY package*.json ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# VITE_OHIF_BASE_URL is baked into the static build at compile time.
# Set this to the OHIF Cloud Run service URL for production builds.
# Example: docker build --build-arg VITE_OHIF_BASE_URL=https://ohif-abc123-uc.a.run.app .
ARG VITE_OHIF_BASE_URL
ENV VITE_OHIF_BASE_URL=$VITE_OHIF_BASE_URL

COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
