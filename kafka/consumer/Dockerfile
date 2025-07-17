# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM mcr.microsoft.com/oss/go/microsoft/golang:1.24.5 AS builder
ARG PROJECT_DIR=/go/src/github.com/microsoft
WORKDIR ${PROJECT_DIR}/confidential-container-demos
COPY /kafka/consumer ./kafka/consumer
COPY /kafka/util ./kafka/util

RUN cd kafka/consumer && CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o consume

FROM mcr.microsoft.com/azurelinux/base/core:3.0
COPY /kafka/consumer/web /web
COPY /kafka/consumer/webtemplates /webtemplates
COPY --from=builder /go/src/github.com/microsoft/confidential-container-demos/kafka/consumer/consume /consume

CMD ["/consume"]
