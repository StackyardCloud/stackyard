FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /vpc-lattice-basic ./examples/vpc-lattice/vpc-lattice-basic
ENTRYPOINT ["/vpc-lattice-basic"]
