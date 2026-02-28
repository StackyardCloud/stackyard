FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /oam-advanced ./examples/oam/oam-advanced
ENTRYPOINT ["/oam-advanced"]
