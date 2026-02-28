FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /dms-advanced ./examples/dms/dms-advanced
ENTRYPOINT ["/dms-advanced"]
