FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /launchwizard ./examples/aws/launchwizard
ENTRYPOINT ["/launchwizard"]
