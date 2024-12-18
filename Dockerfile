FROM golang:1.23 AS foundation

WORKDIR /build
COPY go.mod .
COPY go.sum .
COPY certs certs
RUN go mod download

FROM foundation AS builder

COPY . .
# Build only for linux/amd64 since this is for the container
RUN make

FROM gcr.io/distroless/base AS runtime

COPY --from=builder /build/bin/tracer-linux-* /bin/tracer
COPY --from=builder /build/certs /certs

ENTRYPOINT ["/bin/tracer"]

