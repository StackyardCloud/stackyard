FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /computer-vision-v4.0-preview2023-04-01 ./examples/azure/ai-services/computer-vision-v4.0-preview2023-04-01

ENTRYPOINT ["/computer-vision-v4.0-preview2023-04-01"]
