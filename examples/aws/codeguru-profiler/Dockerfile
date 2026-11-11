FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-profiler ./examples/aws/codeguru-profiler
ENTRYPOINT ["/codeguru-profiler"]
