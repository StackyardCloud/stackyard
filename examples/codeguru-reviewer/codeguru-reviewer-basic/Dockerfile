FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-reviewer-basic ./examples/codeguru-reviewer/codeguru-reviewer-basic
ENTRYPOINT ["/codeguru-reviewer-basic"]
