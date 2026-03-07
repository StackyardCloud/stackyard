FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /orgpolicy-apiv2 ./examples/gcp/orgpolicy-apiv2

ENTRYPOINT ["/orgpolicy-apiv2"]
