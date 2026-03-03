FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecs-advanced ./examples/aws/ecs/ecs-advanced
ENTRYPOINT ["/ecs-advanced"]
