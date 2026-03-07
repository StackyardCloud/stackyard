FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /apikeys-apiv2 ./examples/gcp/apikeys-apiv2
CMD ["/apikeys-apiv2"]
