FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datamigration-2022-03-30-preview-example ./examples/azure/datamigration-2022-03-30-preview

ENTRYPOINT ["/datamigration-2022-03-30-preview-example"]
