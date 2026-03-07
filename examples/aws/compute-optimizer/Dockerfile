FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /compute-optimizer ./examples/aws/compute-optimizer
ENTRYPOINT ["/compute-optimizer"]
