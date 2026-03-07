FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-reviewer ./examples/aws/codeguru-reviewer
ENTRYPOINT ["/codeguru-reviewer"]
