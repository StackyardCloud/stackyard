FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /marketplace-advanced ./examples/marketplace/marketplace-advanced
ENTRYPOINT ["/marketplace-advanced"]
