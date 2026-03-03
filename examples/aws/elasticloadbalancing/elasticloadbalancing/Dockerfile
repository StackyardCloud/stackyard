FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancing ./examples/aws/elasticloadbalancing/elasticloadbalancing
ENTRYPOINT ["/elasticloadbalancing"]
