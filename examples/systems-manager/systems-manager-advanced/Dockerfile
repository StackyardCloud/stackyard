FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /systems-manager-advanced ./examples/systems-manager/systems-manager-advanced
ENTRYPOINT ["/systems-manager-advanced"]
