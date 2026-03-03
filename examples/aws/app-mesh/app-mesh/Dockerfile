FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /app-mesh ./examples/aws/app-mesh/app-mesh
ENTRYPOINT ["/app-mesh"]
