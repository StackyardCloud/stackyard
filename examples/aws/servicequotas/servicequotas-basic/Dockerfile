FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /servicequotas-basic ./examples/aws/servicequotas/servicequotas-basic
ENTRYPOINT ["/servicequotas-basic"]
