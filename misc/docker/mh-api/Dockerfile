FROM 193198941246.dkr.ecr.eu-west-1.amazonaws.com/qnib-golang/metahub-admin:v0.1.1

COPY ./metahub.db /var/lib/metahub.db
COPY ./misc/config/metahub-api.yaml /etc/metahub-api.yaml
CMD ["metahub-admin", "serve"]