# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM mcr.microsoft.com/oss/go/microsoft/golang:1.24.1 AS builder
ARG PROJECT_DIR=/go/src/github.com/microsoft
WORKDIR ${PROJECT_DIR}/confidential-container-demos
COPY /kafka/producer ./kafka/producer
COPY /kafka/util ./kafka/util
RUN cd kafka/producer && CGO_ENABLED=0 GOOS=linux go build -mod=readonly  -v -o produce
FROM mcr.microsoft.com/azurelinux/distroless/minimal:3.0
COPY --from=builder /go/src/github.com/microsoft/confidential-container-demos/kafka/producer/produce /produce
CMD ["/produce"]