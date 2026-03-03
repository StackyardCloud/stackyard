FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ground-station ./examples/aws/ground-station/ground-station
ENTRYPOINT ["/ground-station"]
