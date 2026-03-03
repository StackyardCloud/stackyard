FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /dms-basic ./examples/aws/dms/dms-basic
ENTRYPOINT ["/dms-basic"]
