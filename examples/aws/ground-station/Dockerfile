FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ground-station ./examples/aws/ground-station
ENTRYPOINT ["/ground-station"]
