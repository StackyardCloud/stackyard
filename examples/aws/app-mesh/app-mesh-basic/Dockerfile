FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /app-mesh-basic ./examples/aws/app-mesh/app-mesh-basic
ENTRYPOINT ["/app-mesh-basic"]
