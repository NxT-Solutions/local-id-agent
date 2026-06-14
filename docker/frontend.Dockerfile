FROM node:20-alpine AS builder

WORKDIR /app

RUN corepack enable

COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm --filter localid-react-example... build

FROM nginx:1.27-alpine

COPY docker/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/examples/react/dist /usr/share/nginx/html
COPY docker/frontend-config.template.js /usr/share/nginx/html/config.template.js
COPY docker/frontend-entrypoint.sh /docker-entrypoint.d/40-localid-env.sh

RUN chmod +x /docker-entrypoint.d/40-localid-env.sh

EXPOSE 80
