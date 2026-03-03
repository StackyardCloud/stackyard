FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-profiler ./examples/aws/codeguru-profiler/codeguru-profiler
ENTRYPOINT ["/codeguru-profiler"]
