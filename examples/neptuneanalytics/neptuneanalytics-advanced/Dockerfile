FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptuneanalytics-advanced ./examples/neptuneanalytics/neptuneanalytics-advanced
ENTRYPOINT ["/neptuneanalytics-advanced"]
