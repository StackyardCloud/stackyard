FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /oam-advanced ./examples/aws/oam/oam-advanced
ENTRYPOINT ["/oam-advanced"]
