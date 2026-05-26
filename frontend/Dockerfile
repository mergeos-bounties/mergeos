FROM node:22-alpine AS deps

WORKDIR /app

COPY package*.json ./
RUN npm ci

FROM deps AS build

COPY . .
ENV NODE_ENV=production
RUN npm run build:production

FROM node:22-alpine AS runtime

WORKDIR /app

ENV NODE_ENV=production \
    MERGEOS_ENV=local \
    FRONTEND_HOST=0.0.0.0 \
    FRONTEND_PORT=5173 \
    API_TARGET=http://backend:8080 \
    CLIENT_DIST=dist/client \
    SERVER_ENTRY=dist/server/entry-server.js

COPY package*.json ./
RUN npm ci --omit=dev && npm cache clean --force

COPY server.js ./
COPY --from=build /app/dist ./dist

EXPOSE 5173

CMD ["node", "server.js", "--mode", "production", "--prod"]
