FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecs ./examples/aws/ecs/ecs
ENTRYPOINT ["/ecs"]
