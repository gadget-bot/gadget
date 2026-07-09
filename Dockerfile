FROM golang:1.25-alpine AS builder

WORKDIR /build

# Cache dependencies before copying source
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/gadget

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    adduser -D -s /bin/nologin -H -u 1000 gadget

COPY --from=builder /build/dist/gadget /usr/local/bin/gadget

USER gadget

ENTRYPOINT [ "/usr/local/bin/gadget" ]