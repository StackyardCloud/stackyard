FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /privateca ./examples/aws/privateca/privateca
ENTRYPOINT ["/privateca"]
