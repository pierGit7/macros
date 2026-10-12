# Build stage: static binary, no cgo. Go version matches go.mod.
# Runs on the build machine's own platform and cross-compiles for the target
# (e.g. linux/arm64 for the Raspberry Pi), so no slow emulated Go build.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /macros .
# empty dir for user state; distroless has no shell to mkdir in the runtime stage
RUN mkdir /state

# Runtime stage: binary only. The recipe data is private, so it is not baked into the
# (published) image: mount it at /app/data (read-only). User state lives at DIET_PATH (a volume).
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /macros /app/macros
# owned by nonroot so a fresh volume mounted here inherits a writable owner
COPY --from=build --chown=nonroot:nonroot /state /state
EXPOSE 8080
ENTRYPOINT ["/app/macros"]
