FROM golang:1.25-alpine

WORKDIR /workspace
COPY . .

ENV STACKYARD_ENDPOINT=http://stackyard:4566

RUN go build -o /usr/local/bin/pca-connector-scep ./examples/aws/pca-connector-scep

CMD ["/usr/local/bin/pca-connector-scep"]
