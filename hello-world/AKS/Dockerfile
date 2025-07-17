FROM mcr.microsoft.com/devcontainers/base:dev-ubuntu-24.04 AS build
ENV PROJECT_DIR=~/
WORKDIR ${PROJECT_DIR}/confidential-container-demos/hello-world
COPY . ./
RUN curl -L https://github.com/microsoft/confidential-sidecar-containers/releases/latest/download/verbose-report > verbose-report

FROM mcr.microsoft.com/azurelinux/base/python:3.12
RUN tdnf update -y && tdnf -y upgrade && tdnf -y install nginx && tdnf clean all

WORKDIR /

# copy all files from this folder to working directory (ignores files in .dockerignore)
ARG BUILD_DIR=~/confidential-container-demos/hello-world
COPY --from=build ${BUILD_DIR}/verbose-report /app/verbose-report
COPY --from=build ${BUILD_DIR}/util/. /util/
COPY --from=build ${BUILD_DIR}/AKS/. /app/
COPY --from=build ${BUILD_DIR}/AKS/default /etc/nginx/sites-available/default

WORKDIR /app
RUN chmod +x verbose-report

EXPOSE 80
EXPOSE 443
ENV PYTHONUNBUFFERED=1

RUN rm -rf /etc/nginx/html/
# entrypoint
CMD [ "python3", "main.py"]
