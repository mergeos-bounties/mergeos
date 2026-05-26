FROM node:22-alpine AS build

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
ENV NODE_ENV=production
RUN npm run build

FROM node:22-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --omit=dev

COPY --from=build /app/dist ./dist
COPY server.js ./

ENV NODE_ENV=production
ENV ADMIN_FRONTEND_HOST=0.0.0.0
ENV ADMIN_FRONTEND_PORT=80

EXPOSE 80
CMD ["npm", "run", "production"]
