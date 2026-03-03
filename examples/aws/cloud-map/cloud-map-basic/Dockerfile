FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloud-map-basic ./examples/aws/cloud-map/cloud-map-basic
ENTRYPOINT ["/cloud-map-basic"]
