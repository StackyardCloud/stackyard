FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /launchwizard ./examples/aws/launchwizard/launchwizard
ENTRYPOINT ["/launchwizard"]
