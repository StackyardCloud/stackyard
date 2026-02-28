FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-portal-basic ./examples/singlesignon-portal/singlesignon-portal-basic
ENTRYPOINT ["/singlesignon-portal-basic"]
