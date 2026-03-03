FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /xray ./examples/aws/xray/xray
ENTRYPOINT ["/xray"]
