FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ground-station-advanced ./examples/ground-station/ground-station-advanced
ENTRYPOINT ["/ground-station-advanced"]
