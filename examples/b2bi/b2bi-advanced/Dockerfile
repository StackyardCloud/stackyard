FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /b2bi-advanced ./examples/b2bi/b2bi-advanced
ENTRYPOINT ["/b2bi-advanced"]
