FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /codedeploy ./examples/aws/codedeploy
ENTRYPOINT ["/codedeploy"]
