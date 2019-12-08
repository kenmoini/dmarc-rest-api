FROM golang:alpine

RUN apk add --update --no-cache git curl wget build-base gpgme-dev

RUN go get github.com/kenmoini/dmarc-rest-api

EXPOSE 8080

CMD ["/go/bin/dmarc-rest-api", "-rest-server"]
