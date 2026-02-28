FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mediapackage-advanced ./examples/mediapackage/mediapackage-advanced
ENTRYPOINT ["/mediapackage-advanced"]
