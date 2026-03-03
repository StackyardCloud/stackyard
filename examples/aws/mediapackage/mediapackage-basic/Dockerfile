FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediapackage-basic ./examples/aws/mediapackage/mediapackage-basic
ENTRYPOINT ["/mediapackage-basic"]
