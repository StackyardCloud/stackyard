FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptune-basic ./examples/neptune/neptune-basic
ENTRYPOINT ["/neptune-basic"]
