FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptunedata-advanced ./examples/aws/neptunedata/neptunedata-advanced
ENTRYPOINT ["/neptunedata-advanced"]
