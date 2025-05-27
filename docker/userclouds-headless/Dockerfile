# see https://hub.docker.com/_/debian/tags
FROM debian:12.10

RUN apt update && apt install gnupg2 lsb-release ca-certificates -y
# Adding the hashicorp apt repo - see https://developer.hashicorp.com/terraform/install (linux)
COPY docker/userclouds-common/hashicorp.asc /tmp/hashicorp.asc
RUN gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg /tmp/hashicorp.asc && \
    rm /tmp/hashicorp.asc && \
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/hashicorp.list && \
    apt update && apt install terraform -y

# Adding the current UserClouds repo git hash (plus date and time) as a label.
# To inspect use: docker image inspect --format='{{json .Config.Labels}}' userclouds/docker:latest
ARG CURRENT_REPO_VERSION
LABEL userclouds.version="$CURRENT_REPO_VERSION"

ARG TARGETARCH

COPY bin/linux-$TARGETARCH/. /userclouds/bin/
COPY config/rootdb/container.yaml /userclouds/config/rootdb/
COPY config/idp/container.yaml /userclouds/config/idp/
COPY config/authz/container.yaml /userclouds/config/authz/
COPY config/plex/container.yaml /userclouds/config/plex/
COPY config/base_container.yaml /userclouds/config/

# process routing config file, used by cmd/containerrunner
COPY docker/userclouds-headless/routing.yaml /userclouds/

# Needed for migrations...
COPY config/rootdbstatus/container.yaml /userclouds/config/rootdbstatus/
COPY config/logserver/container.yaml /userclouds/config/logserver/

# needed for provisioning
COPY config/console/container.yaml /userclouds/config/console/

## provisioning files
COPY config/provisioning/dev/company.json /userclouds/provisioning/
COPY config/provisioning/container/tenant_console.json /userclouds/provisioning/tenant_console.json
COPY config/provisioning/samples/company_uc_container_dev.json /userclouds/provisioning/
COPY config/provisioning/samples/tenant_uc_container_dev.json /userclouds/provisioning/
COPY docker/userclouds-headless/run-uc.sh /userclouds/bin/
RUN chmod +x /userclouds/bin/run-uc.sh

RUN mkdir -p /userclouds/logs/
VOLUME /userclouds/logs/

RUN groupadd --gid 99 userclouds && useradd -m --no-log-init --uid 99 --gid 99 userclouds && \
    chown userclouds:userclouds /userclouds/bin && chmod u=rx,go=rx  /userclouds/bin && \
    chown -R userclouds:userclouds /userclouds/logs && chmod --recursive u=rwx,go=rwx  /userclouds/logs && \
    chown userclouds:userclouds /userclouds/provisioning && chmod u=r,go=r  /userclouds/provisioning/*.json && \
    chown userclouds:userclouds /userclouds/config && chmod --recursive u=r,go=r /userclouds/config/*.yaml

USER userclouds:userclouds

######
ENV PATH=/userclouds/bin:$PATH
ENV UC_CONFIG_DIR=/userclouds/config/
ENV UC_UNIVERSE=container
# We don't need a valid region for container env/universe
ENV UC_REGION=""
EXPOSE 3040

ENV UC_CONFIG_MANIFEST_FILE=""
# Where customer will mount the ucconfig manifest file
VOLUME /customer/config/
CMD ["/userclouds/bin/run-uc.sh"]
