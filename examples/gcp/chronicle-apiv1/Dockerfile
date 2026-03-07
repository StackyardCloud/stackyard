FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /chronicle-apiv1 ./examples/gcp/chronicle-apiv1
CMD ["/chronicle-apiv1"]
