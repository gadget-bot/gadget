FROM golang:alpine AS builder

COPY . /usr/local/gadget/

WORKDIR /usr/local/gadget/

RUN go clean && \
    go get && \
    go build -ldflags "-s -w" -o dist/gadget

FROM alpine

COPY --from=builder /usr/local/gadget/dist/gadget /usr/local/bin/gadget

RUN adduser -D -s /bin/nologin -H -u 1000 gadget

USER gadget

ENTRYPOINT [ "/usr/local/bin/gadget" ]
