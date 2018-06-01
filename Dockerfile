FROM centurylink/ca-certs
MAINTAINER Wu Muxian <mw@tectusdreamlab.com>

ADD aws-slack-bot aws-slack-bot

ENTRYPOINT ["/aws-slack-bot"]