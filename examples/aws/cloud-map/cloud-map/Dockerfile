FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloud-map ./examples/aws/cloud-map/cloud-map
ENTRYPOINT ["/cloud-map"]
