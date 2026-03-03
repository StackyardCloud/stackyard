FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /privateca-advanced ./examples/aws/privateca/privateca-advanced
ENTRYPOINT ["/privateca-advanced"]
