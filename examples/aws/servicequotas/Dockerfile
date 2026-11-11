FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /servicequotas ./examples/aws/servicequotas
ENTRYPOINT ["/servicequotas"]
