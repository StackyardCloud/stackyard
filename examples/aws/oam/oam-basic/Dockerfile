FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /oam-basic ./examples/aws/oam/oam-basic
ENTRYPOINT ["/oam-basic"]
