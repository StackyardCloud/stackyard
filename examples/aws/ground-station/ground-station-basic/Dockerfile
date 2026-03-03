FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ground-station-basic ./examples/aws/ground-station/ground-station-basic
ENTRYPOINT ["/ground-station-basic"]
