FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /custom-vision-v3.3 ./examples/azure/ai-services/custom-vision-v3.3

ENTRYPOINT ["/custom-vision-v3.3"]
