FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssm-guiconnect-advanced ./examples/ssm-guiconnect/ssm-guiconnect-advanced
ENTRYPOINT ["/ssm-guiconnect-advanced"]
