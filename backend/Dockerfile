# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.24

FROM golang:${GO_VERSION} AS builder
WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/. ./

ARG SERVICE=worker
RUN test -d "${SERVICE}"
RUN mkdir -p /out
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app "./${SERVICE}"

FROM gcr.io/distroless/base-debian12:latest AS runtime
WORKDIR /app
COPY --from=builder /out/app /app/service
USER nonroot:nonroot
ENTRYPOINT ["/app/service"]
