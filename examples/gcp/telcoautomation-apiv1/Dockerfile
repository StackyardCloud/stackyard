FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /telcoautomation-apiv1 ./examples/gcp/telcoautomation-apiv1

ENTRYPOINT ["/telcoautomation-apiv1"]

