FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /talent-apiv4 ./examples/gcp/talent-apiv4

ENTRYPOINT ["/talent-apiv4"]
