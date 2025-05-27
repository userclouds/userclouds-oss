# see https://hub.docker.com/_/debian/tags
FROM debian:12.10@sha256:00cd074b40c4d99ff0c24540bdde0533ca3791edcdac0de36d6b9fb3260d89e2

# Install ca-certificates, needed in order to be able to talk to AWS APIs
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && apt-get clean && rm -rf /var/lib/apt/lists/*

# Adding the current UserClouds repo git hash (plus date and time) as a label.
# To inspect use: docker image inspect --format='{{json .Config.Labels}}' <image>
ARG CURRENT_REPO_VERSION
LABEL userclouds.version="$CURRENT_REPO_VERSION"
COPY ./bin/linux-amd64/automatedprovisioner /userclouds/bin/
COPY config/provisioning/onprem/tenant_console.json.tmpl userclouds/provisioning/
RUN chmod +x /userclouds/bin/automatedprovisioner

RUN groupadd --gid 99 userclouds && useradd -m --no-log-init --uid 99 --gid 99 userclouds && \
    chown userclouds:userclouds /userclouds/bin/ && chmod u=rx,go=rx  /userclouds/bin && \
    chown userclouds:userclouds /userclouds/provisioning && chmod u=r,go=r /userclouds/provisioning/*.json.tmpl

USER userclouds:userclouds
ENV PATH=/userclouds/bin:$PATH

ENTRYPOINT ["/userclouds/bin/automatedprovisioner"]
