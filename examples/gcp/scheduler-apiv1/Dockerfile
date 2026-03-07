FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /scheduler-apiv1 ./examples/gcp/scheduler-apiv1

ENTRYPOINT ["/scheduler-apiv1"]
