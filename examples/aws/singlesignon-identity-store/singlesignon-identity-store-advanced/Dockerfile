FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-identity-store-advanced ./examples/aws/singlesignon-identity-store/singlesignon-identity-store-advanced
ENTRYPOINT ["/singlesignon-identity-store-advanced"]
