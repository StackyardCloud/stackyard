FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssm-guiconnect-basic ./examples/aws/ssm-guiconnect/ssm-guiconnect-basic
ENTRYPOINT ["/ssm-guiconnect-basic"]
