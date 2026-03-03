FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-portal-basic ./examples/aws/singlesignon-portal/singlesignon-portal-basic
ENTRYPOINT ["/singlesignon-portal-basic"]
