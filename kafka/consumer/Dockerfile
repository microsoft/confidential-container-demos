# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM golang:1.22 as builder
ARG PROJECT_DIR=/go/src/github.com/microsoft
WORKDIR ${PROJECT_DIR}/confidential-container-demos
COPY /kafka/consumer ./kafka/consumer
COPY /kafka/util ./kafka/util
RUN cd kafka/consumer && CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o consume
FROM mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0
COPY /kafka/consumer/web /web
COPY /kafka/consumer/webtemplates /webtemplates
COPY --from=builder /go/src/github.com/microsoft/confidential-container-demos/kafka/consumer/consume /consume
CMD ["/consume"]
