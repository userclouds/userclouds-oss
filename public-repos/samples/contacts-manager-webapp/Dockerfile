# based on https://pythonspeed.com/docker/
# see https://hub.docker.com/_/python/tags
FROM python:3.11.7-bookworm@sha256:a526139339e938726256809dad42ba8dd4edf3133194dbe6f62d400aff6bb886

RUN apt update && apt install gnupg2 curl lsb-release ca-certificates -y
# Adding the hashicorp apt repo - see https://developer.hashicorp.com/terraform/install (linux)
COPY hashicorp.asc /tmp/hashicorp.asc
RUN gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg /tmp/hashicorp.asc && \
    rm /tmp/hashicorp.asc && \
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/hashicorp.list && \
    apt update && apt install terraform -y

# Adding the current UserClouds repo git hash (plus date and time) as a label.
# To inspect use: docker image inspect --format='{{json .Config.Labels}}' userclouds/docker:latest
ARG CURRENT_REPO_VERSION
LABEL userclouds.version="$CURRENT_REPO_VERSION"

WORKDIR /contacts-web-app
ARG ucconfig_version=0.1.12
ARG platform=linux_amd64
RUN curl --fail -L -o /tmp/ucconfig.tar.gz "https://github.com/userclouds/ucconfig/releases/download/v${ucconfig_version}/ucconfig_${ucconfig_version}_${platform}.tar.gz" && \
    mkdir -p /contacts-web-app/bin && \
    tar -xf /tmp/ucconfig.tar.gz -C /contacts-web-app/bin/ ucconfig && \
    rm /tmp/ucconfig.tar.gz && \
    chmod +x /contacts-web-app/bin/ucconfig

RUN pip3 --no-cache-dir install --upgrade pip
COPY requirements.txt requirements.txt
# TODO: it is probably better to utilize a lockfile. but we don't have one yet.
RUN pip3 --no-cache-dir install -r requirements.txt
# TODO: Add non-root used and use it in the container

COPY config/ /contacts-web-app/config
COPY contacts/ /contacts-web-app/contacts
COPY manage.py /contacts-web-app/
VOLUME /contacts-web-app/data

# Note: USERCLOUDS_TENANT_URL USERCLOUDS_CLIENT_ID USERCLOUDS_CLIENT_ID env vars need to be defined when running container

EXPOSE 8000

CMD ["python3", "manage.py", "runserver", "0.0.0.0:8000"]
