# Build the Vite app, then serve the static bundle with nginx.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json ./
# package.json references these via file:../../ui-kit/packages/*, which
# resolves to /ui-kit/packages/* from /app — see additional_contexts in
# docker-compose.yml for where this comes from.
COPY --from=ui-kit . /ui-kit
# Deliberately not copying package-lock.json here: npm's arborist has a
# long-standing bug (crashes with "Cannot read properties of undefined
# (reading 'extraneous')") when reconciling a lockfile against a file:
# dependency whose target lives outside the project root, as ours does.
# Resolving fresh from package.json avoids that code path entirely.
RUN npm install
COPY . .
# .dockerignore excludes .env* from the build context, so this has to come
# in as a build arg instead — Vite forwards any VITE_-prefixed process.env
# var into import.meta.env automatically, no .env file needed.
ARG VITE_FLOWPOS_DASHBOARD_ORIGINS
ENV VITE_FLOWPOS_DASHBOARD_ORIGINS=$VITE_FLOWPOS_DASHBOARD_ORIGINS
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
