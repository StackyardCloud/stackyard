FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /channel-apiv1 ./examples/gcp/channel-apiv1
CMD ["/channel-apiv1"]
