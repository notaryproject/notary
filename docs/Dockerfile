FROM ruby:latest

RUN apt update \
	&& apt install -y --no-install-recommends \
		npm \
	&& rm -rf /var/lib/apt/lists/* \
	&& gem install bundler \
	&& mkdir /www

WORKDIR /www
COPY . /www/

RUN cd /www && bundle install

ENTRYPOINT [ "bundle", "exec", "jekyll", "serve" ]
CMD [ "--incremental", "--host", "0.0.0.0" ]
