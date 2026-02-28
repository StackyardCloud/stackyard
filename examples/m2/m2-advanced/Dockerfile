FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /m2-advanced ./examples/m2/m2-advanced
ENTRYPOINT ["/m2-advanced"]
