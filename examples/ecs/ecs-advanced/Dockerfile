FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecs-advanced ./examples/ecs/ecs-advanced
ENTRYPOINT ["/ecs-advanced"]
