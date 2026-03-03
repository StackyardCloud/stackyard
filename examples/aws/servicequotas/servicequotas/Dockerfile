FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /servicequotas ./examples/aws/servicequotas/servicequotas
ENTRYPOINT ["/servicequotas"]
