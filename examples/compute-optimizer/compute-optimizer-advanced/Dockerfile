FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /compute-optimizer-advanced ./examples/compute-optimizer/compute-optimizer-advanced
ENTRYPOINT ["/compute-optimizer-advanced"]
