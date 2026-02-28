FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /devicefarm-basic ./examples/devicefarm/devicefarm-basic
ENTRYPOINT ["/devicefarm-basic"]
