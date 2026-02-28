FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mpa-advanced ./examples/mpa/mpa-advanced
ENTRYPOINT ["/mpa-advanced"]
