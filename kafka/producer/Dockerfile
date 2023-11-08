# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM golang:1.16 as builder

RUN apt-get -y update && \
   apt-get -y install zip jq libxml2
WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly  -v -o produce
FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/produce /produce
CMD ["/produce"]