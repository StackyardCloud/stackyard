FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /controlcatalog-advanced ./examples/controlcatalog/controlcatalog-advanced
ENTRYPOINT ["/controlcatalog-advanced"]
