FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /cloudtasks-apiv2 ./examples/gcp/cloudtasks-apiv2

ENTRYPOINT ["/cloudtasks-apiv2"]
