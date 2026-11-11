FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptuneanalytics ./examples/aws/neptuneanalytics
ENTRYPOINT ["/neptuneanalytics"]
