#!/bin/bash
# Container runtime configuration script
# Gets secrets config file from S3 and decrypts it using KMS
# This script expects S3URL env variable with the full S3 path to the encrypted config file

if [ -n "$S3URL" ]; then
  echo "Getting config file from S3 (${S3URL}) ..."
  aws --version
  if [[ $? -ne 0 ]]; then
    echo "ERROR: aws-cli not found!"
    exit 1
  fi
  aws --region us-east-1 s3 cp ${S3URL} ./config.encrypted
  aws --region us-east-1 kms decrypt --ciphertext-blob fileb://config.encrypted --output text --query Plaintext | base64 -d > app/config/config.json
  rm -f config.encrypted
else
  echo "ERROR: S3URL variable not set!"
  exit 1
fi

