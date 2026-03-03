FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloud-map-advanced ./examples/aws/cloud-map/cloud-map-advanced
ENTRYPOINT ["/cloud-map-advanced"]
