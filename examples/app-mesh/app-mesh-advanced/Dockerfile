FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /app-mesh-advanced ./examples/app-mesh/app-mesh-advanced
ENTRYPOINT ["/app-mesh-advanced"]
