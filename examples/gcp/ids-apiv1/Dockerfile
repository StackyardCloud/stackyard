FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /ids-apiv1 ./examples/gcp/ids-apiv1

ENTRYPOINT ["/ids-apiv1"]
