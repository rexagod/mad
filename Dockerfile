FROM golang:latest as builder

WORKDIR /

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN make mad

FROM ubuntu:latest

RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /

COPY --from=builder /mad .

# Append "-v=X" to set the verbosity level.
CMD ["./mad"]
