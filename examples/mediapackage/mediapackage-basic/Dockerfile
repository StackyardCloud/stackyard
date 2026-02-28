FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediapackage-basic ./examples/mediapackage/mediapackage-basic
ENTRYPOINT ["/mediapackage-basic"]
