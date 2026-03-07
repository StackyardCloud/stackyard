FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptuneanalytics ./examples/aws/neptuneanalytics
ENTRYPOINT ["/neptuneanalytics"]
