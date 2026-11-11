FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /chronicle-apiv1 ./examples/gcp/chronicle-apiv1
CMD ["/chronicle-apiv1"]
