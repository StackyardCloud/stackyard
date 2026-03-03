FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-identity-store-basic ./examples/aws/singlesignon-identity-store/singlesignon-identity-store-basic
ENTRYPOINT ["/singlesignon-identity-store-basic"]
