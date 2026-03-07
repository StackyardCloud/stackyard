FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /oam ./examples/aws/oam
ENTRYPOINT ["/oam"]
