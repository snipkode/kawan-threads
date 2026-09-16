# syntax=docker/dockerfile:1

# ------------------------------------------------------------------
# Stage 1: build the Vite/React app
# ------------------------------------------------------------------
FROM node:20-alpine AS build
WORKDIR /app

COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY web/ ./
RUN npm run build

# ------------------------------------------------------------------
# Stage 2: nginx serving the SPA + proxying /api to the api container
# ------------------------------------------------------------------
FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]