FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /marketplace ./examples/aws/marketplace/marketplace
ENTRYPOINT ["/marketplace"]
