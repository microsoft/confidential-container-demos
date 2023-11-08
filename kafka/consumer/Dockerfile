# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM golang:1.21 as builder

#RUN apt-get -y update && \
#   apt-get -y install zip jq libxml2

# Copy local code to the container image.
WORKDIR /app

# Retrieve application dependencies using go modules.
# Allows container builds to reuse downloaded dependencies.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly  -v -o consume

FROM alpine:3
RUN apk add --no-cache ca-certificates

COPY web /web
COPY webtemplates /webtemplates
COPY --from=builder /app/consume /consume

CMD ["/consume"]