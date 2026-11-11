FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /optimization-apiv1 ./examples/gcp/optimization-apiv1

ENTRYPOINT ["/optimization-apiv1"]
