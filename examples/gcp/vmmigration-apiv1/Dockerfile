FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /vmmigration-apiv1 ./examples/gcp/vmmigration-apiv1

ENTRYPOINT ["/vmmigration-apiv1"]
