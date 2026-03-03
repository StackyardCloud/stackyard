FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /launchwizard-advanced ./examples/aws/launchwizard/launchwizard-advanced
ENTRYPOINT ["/launchwizard-advanced"]
