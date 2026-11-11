FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /controlcatalog ./examples/aws/controlcatalog
ENTRYPOINT ["/controlcatalog"]
