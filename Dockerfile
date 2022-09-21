FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -a -installsuffix cgo

FROM alpine
COPY --from=builder /app/aws-check-alarms /aws-check-alarms
ENTRYPOINT ["/aws-check-alarms"]