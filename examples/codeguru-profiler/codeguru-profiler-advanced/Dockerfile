FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-profiler-advanced ./examples/codeguru-profiler/codeguru-profiler-advanced
ENTRYPOINT ["/codeguru-profiler-advanced"]
