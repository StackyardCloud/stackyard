FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /launchwizard-basic ./examples/launchwizard/launchwizard-basic
ENTRYPOINT ["/launchwizard-basic"]
