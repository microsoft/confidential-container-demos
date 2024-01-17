FROM ubuntu:22.04
RUN apt-get update && apt-get -y upgrade && apt-get -y install nginx python3.10

WORKDIR /app

# copy all files from this folder to working directory (ignores files in .dockerignore)
COPY . /app/
RUN mv default /etc/nginx/sites-available/default

EXPOSE 80
EXPOSE 443
ENV PYTHONUNBUFFERED=1

RUN rm -rf /usr/share/nginx/html/
# entrypoint
CMD [ "python3.10", "main.py"]
