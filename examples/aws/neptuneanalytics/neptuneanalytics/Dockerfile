FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptuneanalytics ./examples/aws/neptuneanalytics/neptuneanalytics
ENTRYPOINT ["/neptuneanalytics"]
