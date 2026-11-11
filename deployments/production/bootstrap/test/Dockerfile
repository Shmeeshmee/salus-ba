# The backup test's image: the runner (age, the SFTP client, curl, the Docker CLI against the host's daemon) and the
# throwaway SFTP server (sshd allowing SFTP only, as a Storage Box). Pinned by digest.
FROM docker:29.9.3-cli@sha256:2357639ad1ea7d0194134bf3292622fd4e562483966ed674e5d7c945b5f3b28d AS docker-cli

FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852
RUN apt-get update -qq \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends age curl openssh-client openssh-server \
	&& rm -rf /var/lib/apt/lists/* \
	&& useradd --create-home --shell /usr/sbin/nologin storagebox \
	&& usermod -p '*' storagebox \
	&& mkdir -p /run/sshd
COPY --from=docker-cli /usr/local/bin/docker /usr/local/bin/docker
COPY sftp-server.conf /etc/ssh/sshd_config.d/00-test-sftp.conf
