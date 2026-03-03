FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediapackage ./examples/aws/mediapackage/mediapackage
ENTRYPOINT ["/mediapackage"]
