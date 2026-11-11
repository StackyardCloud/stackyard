FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /telcoautomation-apiv1 ./examples/gcp/telcoautomation-apiv1

ENTRYPOINT ["/telcoautomation-apiv1"]

