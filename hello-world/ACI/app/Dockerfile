FROM mcr.microsoft.com/devcontainers/base:dev-ubuntu-24.04 AS build
ENV PROJECT_DIR=~/
WORKDIR ${PROJECT_DIR}/confidential-container-demos/hello-world/ACI/app
COPY . ./
RUN curl -L https://github.com/microsoft/confidential-sidecar-containers/releases/latest/download/verbose-report > verbose-report

FROM mcr.microsoft.com/azurelinux/base/python:3.12
RUN pip install flask --no-cache-dir

# all commands start from this directory
WORKDIR /app

# copy all files from this folder to working directory (ignores files in .dockerignore)
ARG BUILD_DIR=~/confidential-container-demos/hello-world/ACI/app
COPY --from=build ${BUILD_DIR}/Dockerfile ${BUILD_DIR}/main.py ${BUILD_DIR}/verbose-report /app/

RUN chmod +x verbose-report

EXPOSE 80
ENV PYTHONUNBUFFERED=1

# set the start command
CMD [ "python3", "main.py" ]
