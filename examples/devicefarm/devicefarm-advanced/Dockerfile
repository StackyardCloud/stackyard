FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /devicefarm-advanced ./examples/devicefarm/devicefarm-advanced
ENTRYPOINT ["/devicefarm-advanced"]
