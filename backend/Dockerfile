# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
# Bind source only for this build step to keep intermediate images small.
RUN --mount=type=bind,target=/src --mount=type=secret,id=proxy_ca \
    --mount=type=tmpfs,target=/tmp --mount=type=tmpfs,target=/root/.cache/go-build \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    CGO_ENABLED=0 go build -mod=readonly -trimpath -o /server ./cmd/server && rm -rf /go/pkg/mod
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
