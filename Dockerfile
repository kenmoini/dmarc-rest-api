FROM golang:alpine

RUN mkdir -p /opt/dmarc-rest-api

WORKDIR /opt/dmarc-rest-api

RUN go get github.com/kenmoini/dmarc-rest-api

CMD ["./dmarc-rest-api" "-rest-server"]