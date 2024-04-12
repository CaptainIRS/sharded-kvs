FROM golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

# Prevent GLIBC compatibility issues
ENV CGO_ENABLED=0

COPY cmd/ cmd/
COPY internal/ internal/

RUN go build -o /app cmd/node/main.go


FROM gcr.io/distroless/base

COPY --from=builder /app/main /main

CMD ["/main", "-port", "8080"]
