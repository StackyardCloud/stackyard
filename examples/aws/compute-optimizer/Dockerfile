FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /compute-optimizer ./examples/aws/compute-optimizer
ENTRYPOINT ["/compute-optimizer"]
