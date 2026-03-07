FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /rapidmigrationassessment-apiv1 ./examples/gcp/rapidmigrationassessment-apiv1

ENTRYPOINT ["/rapidmigrationassessment-apiv1"]
