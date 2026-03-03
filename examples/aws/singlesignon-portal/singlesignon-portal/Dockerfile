FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-portal ./examples/aws/singlesignon-portal/singlesignon-portal
ENTRYPOINT ["/singlesignon-portal"]
