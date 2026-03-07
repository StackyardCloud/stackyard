FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloud-map ./examples/aws/cloud-map
ENTRYPOINT ["/cloud-map"]
