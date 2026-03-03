FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /compute-optimizer ./examples/aws/compute-optimizer/compute-optimizer
ENTRYPOINT ["/compute-optimizer"]
