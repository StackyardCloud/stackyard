FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ecs-basic ./examples/ecs/ecs-basic
ENTRYPOINT ["/ecs-basic"]
