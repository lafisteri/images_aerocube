FROM browsers/base:7.4.2

ARG VERSION
ARG PACKAGE=google-chrome-stable
ARG INSTALL_DIR=chrome

LABEL browser=$PACKAGE:$VERSION

RUN \
        curl -s https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - && \
        echo 'deb [arch=amd64] https://dl.google.com/linux/chrome/deb/ stable main' > /etc/apt/sources.list.d/google.list && \
        apt-get update && \
        CHROME_DEB_VERSION="$VERSION" && \
        if ! apt-cache madison "$PACKAGE" | awk '{print $3}' | awk -v version="$CHROME_DEB_VERSION" '$0 == version { found = 1 } END { exit !found }'; then \
          CHROME_BUILD_VERSION="${VERSION%%-*}" && \
          CHROME_DEB_VERSION="$(apt-cache madison "$PACKAGE" | awk '{print $3}' | awk -v version_prefix="${CHROME_BUILD_VERSION}-" 'index($0, version_prefix) == 1 { print; exit }')" && \
          test -n "$CHROME_DEB_VERSION"; \
        fi && \
        apt-get -y --no-install-recommends install iproute2 "$PACKAGE=$CHROME_DEB_VERSION" && \
         ( \
          sed -i -e 's@exec -a "$0" "$HERE/chrome"@& --no-sandbox --disable-gpu@' /opt/google/$INSTALL_DIR/$PACKAGE || \
          sed -i -e 's@exec -a "$0" "$HERE/chrome"@& --no-sandbox --disable-gpu@' /opt/google/$INSTALL_DIR/google-chrome \
         ) && \
        chown root:root /opt/google/$INSTALL_DIR/chrome-sandbox && \
        chmod 4755 /opt/google/$INSTALL_DIR/chrome-sandbox && \
        google-chrome --version && \
        rm -Rf /tmp/* && rm -Rf /var/lib/apt/lists/*
