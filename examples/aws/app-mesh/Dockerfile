FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /app-mesh ./examples/aws/app-mesh
ENTRYPOINT ["/app-mesh"]
