FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptuneanalytics-basic ./examples/aws/neptuneanalytics/neptuneanalytics-basic
ENTRYPOINT ["/neptuneanalytics-basic"]
