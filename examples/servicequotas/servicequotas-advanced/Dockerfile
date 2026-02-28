FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /servicequotas-advanced ./examples/servicequotas/servicequotas-advanced
ENTRYPOINT ["/servicequotas-advanced"]
