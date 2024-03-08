FROM golang AS builder

# create a working directory inside the image
WORKDIR /source

# copy Go modules and dependencies to image
COPY ./cmd .
COPY ./pkg .
COPY ./go.mod .
COPY ./go.sum .

# download Go modules and dependencies
RUN go mod download

# compile application
RUN CGO_ENABLED=0 go build -a -o /app/bot /source/cmd/bot/main.go

# Now copy it into a base image.
FROM alpine

# Create a group and user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Tell docker that all future commands should run as the appuser user
USER appuser

COPY --from=builder /app/bot /app/bot
CMD ["/app/bot"]