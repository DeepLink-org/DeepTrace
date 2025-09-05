FROM ubuntu:22.04
LABEL maintainer="The Authors <liqilong@pjlab.org.cn>"

ENV TZ=Asia/Shanghai
RUN ln -fs /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone

RUN apt-get update && apt-get install -y \
    openssh-server \
    vim \
    tzdata \
    systemd \
    bash \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /var/run/sshd && \
    echo 'root:change_here' | chpasswd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config

COPY --chown=root:root ./pushgateway /bin/pushgateway

EXPOSE 9091
RUN mkdir -p /pushgateway && chown root:root /pushgateway
WORKDIR /pushgateway

ENTRYPOINT [ "/bin/pushgateway" ]