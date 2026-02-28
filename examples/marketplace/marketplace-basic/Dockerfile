FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /marketplace-basic ./examples/marketplace/marketplace-basic
ENTRYPOINT ["/marketplace-basic"]
