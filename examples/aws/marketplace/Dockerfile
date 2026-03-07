FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /marketplace ./examples/aws/marketplace
ENTRYPOINT ["/marketplace"]
