FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go build -o server .

# Simulate a slow build step (e.g. a heavy asset pipeline, codegen, etc.)
RUN echo "Running slow build step..." && sleep 30 && echo "Done."

# ---

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

ENTRYPOINT ["./server"]
