FROM golang:alpine

RUN apk add --update --no-cache git curl wget build-base gpgme-dev && \
    addgroup -g 1000 gouser && \
    adduser -S -D -u 1000 -G gouser gouser && \
    chown -R gouser:gouser /home/gouser

USER gouser

RUN go get github.com/kenmoini/dmarc-rest-api

EXPOSE 8080

CMD ["/go/bin/dmarc-rest-api", "-rest-server"]
