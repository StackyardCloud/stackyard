FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataplex-apiv1 ./examples/gcp/dataplex-apiv1

ENTRYPOINT ["/dataplex-apiv1"]
