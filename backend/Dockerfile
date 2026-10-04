# Build stage: static binary, no cgo. Go version matches go.mod.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /macros .
# empty dir for user state; distroless has no shell to mkdir in the runtime stage
RUN mkdir /state

# Runtime stage: binary + read-only seed data. User state lives at DIET_PATH (a volume).
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /macros /app/macros
COPY data/ricette.json data/macro-sources.json /app/data/
# owned by nonroot so a fresh volume mounted here inherits a writable owner
COPY --from=build --chown=nonroot:nonroot /state /state
EXPOSE 8080
ENTRYPOINT ["/app/macros"]
