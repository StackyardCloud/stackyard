FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-oidc ./examples/aws/singlesignon-oidc/singlesignon-oidc
ENTRYPOINT ["/singlesignon-oidc"]

