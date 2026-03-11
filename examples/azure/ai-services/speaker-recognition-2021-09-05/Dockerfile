FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /speaker-recognition-2021-09-05 ./examples/azure/ai-services/speaker-recognition-2021-09-05

ENTRYPOINT ["/speaker-recognition-2021-09-05"]
