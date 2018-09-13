FROM node:8-alpine

ARG ARCH=x86_64

# install chromium/firefox for headless browser tests
ENV CHROME_BIN=/usr/bin/browser

RUN case "${ARCH}" in                                      \
        amd64)                                             \
          apk add --no-cache chromium udev ttf-freefont;   \
          ln -s /usr/bin/chromium-browser /usr/bin/browser \
          ;;                                               \
        ppc64le)                                           \
          apk add --no-cache firefox-esr udev ttf-freefont python build-base; \
          ln -s /usr/bin/firefox /usr/bin/browser          \
          ;;                                               \
    esac

# set workdir
WORKDIR /ffdl/ffdl-dashboard

# fix npm permission issues (https://github.com/creationix/nvm/issues/1407#issuecomment-316858947)
RUN npm config set user 0
RUN npm config set unsafe-perm true
# install global dependencies
# RUN npm install --quiet -g @angular/cli
RUN npm install --quiet -g @angular/cli@6.0.8
RUN npm install --quiet -g http-server

# set non-root user
RUN chown -R node:node .
USER node

# install local dependencies
ADD package.json ./
RUN npm install --quiet

# add source files
ADD .angular-cli.json .editorconfig karma.conf.js package-lock.json \
  protractor.conf.js README.md tsconfig.json tslint.json ./
ADD e2e/ ./e2e/
ADD src/ ./src/

# create production build
RUN ng build --prod

# clean up (make sure to squash the image afterwards!)
RUN rm -r ./node_modules /home/node/./.npm

EXPOSE 8080

CMD ["http-server", "./dist"]
