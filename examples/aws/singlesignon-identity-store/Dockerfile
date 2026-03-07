FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-identity-store ./examples/aws/singlesignon-identity-store
ENTRYPOINT ["/singlesignon-identity-store"]
