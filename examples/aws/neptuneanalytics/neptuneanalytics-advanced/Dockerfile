FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptuneanalytics-advanced ./examples/aws/neptuneanalytics/neptuneanalytics-advanced
ENTRYPOINT ["/neptuneanalytics-advanced"]
