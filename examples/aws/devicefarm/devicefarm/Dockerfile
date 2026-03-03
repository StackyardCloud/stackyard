FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /devicefarm ./examples/aws/devicefarm/devicefarm
ENTRYPOINT ["/devicefarm"]
