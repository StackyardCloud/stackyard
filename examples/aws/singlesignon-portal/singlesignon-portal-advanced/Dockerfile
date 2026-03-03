FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-portal-advanced ./examples/aws/singlesignon-portal/singlesignon-portal-advanced
ENTRYPOINT ["/singlesignon-portal-advanced"]
