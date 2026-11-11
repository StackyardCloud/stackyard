FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /oracledatabase-apiv1 ./examples/gcp/oracledatabase-apiv1

ENTRYPOINT ["/oracledatabase-apiv1"]
