FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/taxsettings ./examples/aws/taxsettings

CMD ["/usr/local/bin/taxsettings"]
