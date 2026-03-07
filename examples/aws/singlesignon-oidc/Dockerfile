FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-oidc ./examples/aws/singlesignon-oidc
ENTRYPOINT ["/singlesignon-oidc"]

