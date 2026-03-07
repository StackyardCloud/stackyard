FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /trace-apiv1 ./examples/gcp/trace-apiv1

ENTRYPOINT ["/trace-apiv1"]
