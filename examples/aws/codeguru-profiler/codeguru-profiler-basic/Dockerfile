FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-profiler-basic ./examples/aws/codeguru-profiler/codeguru-profiler-basic
ENTRYPOINT ["/codeguru-profiler-basic"]
