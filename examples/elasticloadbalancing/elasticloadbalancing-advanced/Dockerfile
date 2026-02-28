FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancing-advanced ./examples/elasticloadbalancing/elasticloadbalancing-advanced
ENTRYPOINT ["/elasticloadbalancing-advanced"]
