FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /servicequotas-advanced ./examples/aws/servicequotas/servicequotas-advanced
ENTRYPOINT ["/servicequotas-advanced"]
