FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-oidc-basic ./examples/aws/singlesignon-oidc/singlesignon-oidc-basic
ENTRYPOINT ["/singlesignon-oidc-basic"]

