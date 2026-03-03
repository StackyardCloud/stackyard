FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /controlcatalog ./examples/aws/controlcatalog/controlcatalog
ENTRYPOINT ["/controlcatalog"]
