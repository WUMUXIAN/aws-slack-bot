FROM centurylink/ca-certs
MAINTAINER Wu Muxian <mw@tectusdreamlab.com>

ADD slack_webhook_url slack_webhook_url 
ADD aws-slack-bot aws-slack-bot

ENTRYPOINT ["/aws-slack-bot"]