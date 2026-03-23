# Build stage
FROM elixir:1.18-alpine AS build

RUN apk add --no-cache build-base git python3

WORKDIR /app

# Install hex + rebar
RUN mix local.hex --force && mix local.rebar --force

# Set build env
ENV MIX_ENV=prod

# Copy mix files first for dependency caching
COPY mix.exs mix.lock ./
RUN mix deps.get --only prod
RUN mix deps.compile

# Copy app source
COPY config config
COPY lib lib
COPY assets assets
COPY priv priv

# Build assets (tailwind + esbuild)
RUN mix assets.deploy

# Compile and build release
RUN mix compile
RUN mix release

# Runtime stage
FROM alpine:3.20 AS runtime

RUN apk add --no-cache libstdc++ openssl ncurses-libs

WORKDIR /app

# Copy release from build stage
COPY --from=build /app/_build/prod/rel/htmlgraph_dashboard ./

# Create storage directory for mounted volumes
RUN mkdir -p /storage

# Expose port
EXPOSE 4000

# Set env vars
ENV HTMLGRAPH_WORKSPACE=/workspace
ENV HTMLGRAPH_DB_PATH=/workspace/.htmlgraph/htmlgraph.db
ENV PHX_HOST=localhost
ENV PORT=4000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -q --spider http://localhost:4000/up || exit 1

# Start the release
CMD ["bin/htmlgraph_dashboard", "start"]
