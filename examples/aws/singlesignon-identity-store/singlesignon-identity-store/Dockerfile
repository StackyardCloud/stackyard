FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-identity-store ./examples/aws/singlesignon-identity-store/singlesignon-identity-store
ENTRYPOINT ["/singlesignon-identity-store"]
