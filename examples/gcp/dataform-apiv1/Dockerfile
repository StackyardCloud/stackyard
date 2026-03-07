FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataform-apiv1 ./examples/gcp/dataform-apiv1

ENTRYPOINT ["/dataform-apiv1"]
