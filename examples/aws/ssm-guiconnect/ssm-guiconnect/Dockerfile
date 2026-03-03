FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssm-guiconnect ./examples/aws/ssm-guiconnect/ssm-guiconnect
ENTRYPOINT ["/ssm-guiconnect"]
