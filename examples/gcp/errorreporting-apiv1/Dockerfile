FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /errorreporting-apiv1 ./examples/gcp/errorreporting-apiv1

ENTRYPOINT ["/errorreporting-apiv1"]
