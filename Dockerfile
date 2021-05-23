FROM golang:alpine AS builder

COPY . /usr/local/gadget/

WORKDIR /usr/local/gadget/

RUN go clean && \
    go get && \
    go build -o dist/gadget

FROM busybox

COPY --from=builder /usr/local/gadget/dist/gadget /usr/local/bin/gadget

RUN adduser -D -s /bin/nologin -H -u 1000 gadget

USER gadget

ENTRYPOINT [ "/usr/local/bin/gadget" ]
