# syntax=docker/dockerfile:1

# Build the static server binary from pre-generated sources.
# Generated code (internal/**/*.gen.go) must already be present in the build
# context: CI runs `make generate` before `docker build`.
FROM golang:1.26-alpine AS build
# Use exactly the image's toolchain; fail loudly if it ever skews from go.mod.
ENV GOTOOLCHAIN=local
WORKDIR /src

# Download modules first so this layer caches across source-only changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

# Minimal, non-root runtime image.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
