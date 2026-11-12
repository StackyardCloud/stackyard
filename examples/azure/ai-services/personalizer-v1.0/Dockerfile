FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /personalizer-v1.0 ./examples/azure/ai-services/personalizer-v1.0

ENTRYPOINT ["/personalizer-v1.0"]
