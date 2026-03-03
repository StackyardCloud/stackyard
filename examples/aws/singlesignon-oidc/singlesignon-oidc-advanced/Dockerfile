FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-oidc-advanced ./examples/aws/singlesignon-oidc/singlesignon-oidc-advanced
ENTRYPOINT ["/singlesignon-oidc-advanced"]

