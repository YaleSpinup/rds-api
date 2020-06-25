#!/bin/bash
# Container runtime configuration script
# Gets encrypted config file from SSM parameter store
# This script expects SSMPATH env variable with the full SSMPATH path to the encrypted config file

if [ -n "$SSMPATH" ]; then
  echo "Getting config file from SSM Parameter Store (${SSMPATH}) ..."
  aws --version
  if [[ $? -ne 0 ]]; then
    echo "ERROR: awscli not found!"
    exit 1
  fi
  aws --region us-east-1 ssm get-parameter --name "${SSMPATH}" --with-decryption --output text --query "Parameter.Value" | base64 -d > config/config.json
else
  echo "ERROR: SSMPATH variable not set!"
  exit 1
fi
