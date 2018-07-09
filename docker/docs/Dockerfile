FROM alpine:3.4

ENV HUGO_VERSION 0.42.2
ENV HUGO_TARGZ hugo_${HUGO_VERSION}_linux-64bit


# Install pygments (for syntax highlighting)
RUN apk update && apk add bash \
    git \
    py-pygments \
    curl \
    && rm -rf /var/cache/apk/*

# Download and Install hugo
RUN curl -L https://github.com/spf13/hugo/releases/download/v${HUGO_VERSION}/${HUGO_TARGZ}.tar.gz -o /usr/local/${HUGO_TARGZ}.tar.gz && \
    tar xzf /usr/local/${HUGO_TARGZ}.tar.gz -C /usr/local/bin/ \
	&& rm /usr/local/${HUGO_TARGZ}.tar.gz

# Create user
ARG uid=1000
ARG gid=1000
RUN addgroup -g $gid hugo
RUN adduser -D -u $uid -G hugo hugo

RUN mkdir -p /docs && \
    chown hugo:hugo -R /docs

USER hugo
WORKDIR /docs

EXPOSE 1313
CMD hugo version
