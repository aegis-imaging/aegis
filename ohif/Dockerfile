FROM ohif/app:latest

# gettext provides envsubst for API_URL substitution at container start
USER root
RUN apk add --no-cache gettext

# Override nginx config: listen on 8080 (Cloud Run default) + add /health
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Config template — API_URL placeholder substituted by entrypoint
COPY app-config.js.template /app-config.js.template

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
