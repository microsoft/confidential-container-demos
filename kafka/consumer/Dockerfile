# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM golang:1.21 as builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly  -v -o consume
FROM mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0
COPY web /web
COPY webtemplates /webtemplates
COPY --from=builder /app/consume /consume
CMD ["/consume"]