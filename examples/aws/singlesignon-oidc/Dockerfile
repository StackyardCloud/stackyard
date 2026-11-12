FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-oidc ./examples/aws/singlesignon-oidc
ENTRYPOINT ["/singlesignon-oidc"]

