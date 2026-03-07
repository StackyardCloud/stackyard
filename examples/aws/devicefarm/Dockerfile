FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /devicefarm ./examples/aws/devicefarm
ENTRYPOINT ["/devicefarm"]
