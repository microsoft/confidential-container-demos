FROM mcr.microsoft.com/devcontainers/base:dev-ubuntu-24.04 AS build
ENV PROJECT_DIR=~/
WORKDIR ${PROJECT_DIR}/confidential-container-demos/hello-world
COPY . ./
RUN curl -L https://github.com/microsoft/confidential-sidecar-containers/releases/latest/download/verbose-report > verbose-report

FROM mcr.microsoft.com/azurelinux/base/python:3.12
RUN pip install flask --no-cache-dir

# all commands start from this directory
WORKDIR /

# copy all files from this folder to working directory (ignores files in .dockerignore)
ARG BUILD_DIR=~/confidential-container-demos/hello-world
COPY --from=build ${BUILD_DIR}/verbose-report /app/verbose-report
COPY --from=build ${BUILD_DIR}/util/. /util/
COPY --from=build ${BUILD_DIR}/ACI/. /app/

WORKDIR /app

RUN chmod +x verbose-report

EXPOSE 80
ENV PYTHONUNBUFFERED=1

# set the start command
CMD [ "python3", "main.py" ]
