FROM node:22-alpine AS build

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY tsconfig.json vite.config.ts index.html ./
COPY src/ src/
RUN npm run build

FROM node:22-alpine AS serve

WORKDIR /app
COPY --from=build /app/dist dist/
COPY server.mjs ./

ENV PORT=3000
ENV API_PROXY_TARGET=http://api-service:8080

USER node
EXPOSE 3000
CMD ["node", "server.mjs"]
