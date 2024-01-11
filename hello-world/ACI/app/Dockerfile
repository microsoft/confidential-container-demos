FROM alpine:3.19
RUN apk update && apk add python3 py3-pip --no-cache
RUN pip3 install flask --no-cache-dir --break-system-packages

# all commands start from this directory
WORKDIR /app

# copy all files from this folder to working directory (ignores files in .dockerignore)
COPY Dockerfile main.py verbose-report /app/

EXPOSE 80
ENV PYTHONUNBUFFERED=1

# set the start command
CMD [ "python3", "main.py" ]
