FROM node:20-alpine AS deps

WORKDIR /app
COPY package.json ./
RUN npm install

# Étape de build
FROM node:20-alpine AS builder

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NEXT_TELEMETRY_DISABLED=1
# BACKEND_URL est passé comme build ARG pour être baked dans next build
ARG BACKEND_URL=http://host.docker.internal:8083
ENV BACKEND_URL=$BACKEND_URL
RUN npm run build

# Image finale de production
FROM node:20-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000

CMD ["node", "server.js"]
