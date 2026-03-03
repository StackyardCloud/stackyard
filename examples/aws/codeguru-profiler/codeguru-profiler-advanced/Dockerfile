FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-profiler-advanced ./examples/aws/codeguru-profiler/codeguru-profiler-advanced
ENTRYPOINT ["/codeguru-profiler-advanced"]
