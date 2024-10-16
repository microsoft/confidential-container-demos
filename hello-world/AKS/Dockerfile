FROM mcr.microsoft.com/devcontainers/base:dev-ubuntu-24.04 AS build
ENV PROJECT_DIR=~/
WORKDIR ${PROJECT_DIR}/confidential-container-demos
COPY . ./
RUN chmod +x hello-world/get_verbose_report.sh && ./hello-world/get_verbose_report.sh

FROM mcr.microsoft.com/azurelinux/base/python:3.12
RUN tdnf update -y && tdnf -y upgrade && tdnf -y install nginx && tdnf clean all

WORKDIR /app

# copy all files from this folder to working directory (ignores files in .dockerignore)
ARG BUILD_DIR=~/confidential-container-demos/hello-world/AKS
COPY --from=build ${BUILD_DIR}/. /app/
COPY --from=build ${BUILD_DIR}/default /etc/nginx/sites-available/default

RUN chmod +x verbose-report

EXPOSE 80
EXPOSE 443
ENV PYTHONUNBUFFERED=1

RUN rm -rf /etc/nginx/html/
# entrypoint
CMD [ "python3", "main.py"]
