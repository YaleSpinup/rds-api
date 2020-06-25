#!/bin/bash
# Container runtime configuration script
# Gets secrets config file from SSM parameter store and uses Deco to substitute parameter values
# This script expects SSMPATH env variable with the full SSMPATH path to the encrypted config file

if [ -n "$SSMPATH" ]; then
  echo "Getting config file from SSM Parameter Store (${SSMPATH}) ..."
  aws --version
  if [[ $? -ne 0 ]]; then
    echo "ERROR: aws-cli not found!"
    exit 1
  fi
  aws --region us-east-1 ssm get-parameter --name "${SSMPATH}" --with-decryption --output text --query Parameter.Value | base64 -d > deco-config.json
  deco validate deco-config.json || exit 1
  deco run deco-config.json
  rm -f deco-config.json config.encrypted
else
  echo "ERROR: SSMPATH variable not set!"
  exit 1
fi
