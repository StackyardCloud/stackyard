FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /notebooks-apiv1 ./examples/gcp/notebooks-apiv1

ENTRYPOINT ["/notebooks-apiv1"]
