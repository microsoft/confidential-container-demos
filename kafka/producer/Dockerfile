# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

FROM golang:1.22.2 as builder
ARG PROJECT_DIR=/go/src/github.com/microsoft
WORKDIR ${PROJECT_DIR}/confidential-container-demos
COPY /kafka/producer ./kafka/producer
COPY /kafka/util ./kafka/util
RUN cd kafka/producer && CGO_ENABLED=0 GOOS=linux go build -mod=readonly  -v -o produce
FROM mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0
COPY --from=builder /go/src/github.com/microsoft/confidential-container-demos/kafka/producer/produce /produce
CMD ["/produce"]