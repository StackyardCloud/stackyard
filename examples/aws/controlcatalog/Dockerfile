FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /controlcatalog ./examples/aws/controlcatalog
ENTRYPOINT ["/controlcatalog"]
