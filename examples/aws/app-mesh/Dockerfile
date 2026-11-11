FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /app-mesh ./examples/aws/app-mesh
ENTRYPOINT ["/app-mesh"]
