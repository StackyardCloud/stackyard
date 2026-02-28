FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-profiler-basic ./examples/codeguru-profiler/codeguru-profiler-basic
ENTRYPOINT ["/codeguru-profiler-basic"]
