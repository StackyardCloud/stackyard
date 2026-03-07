FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /meet-apiv2 ./examples/gcp/meet-apiv2
CMD ["/meet-apiv2"]
