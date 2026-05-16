#!/usr/bin/env bash
export STACK=lioazure
export PROJECT=test
export DEFANG_EVENTS_UPLOAD_URL=file:///tmp/events-azure.json
export STATES_EVENTS_UPLOAD_URL=file:///tmp/states-azure.json
export DEFANG_PULUMI_DIFF=1

export AZURE_SUBSCRIPTION_ID=f311c4db-e998-4c94-906c-7e2637303a05
export AZURE_LOCATION=westus
export AZURE_TENANT_ID=12c0f515-fa47-402f-9982-de7646d3cb28
export AZURE_CLIENT_ID=80fc261c-daa9-4d91-86f7-74c110dac086

go run -race . "${1-preview}" "${2-IlpzZXJ2aWNlczoKICBuZ2lueDoKICAgIGltYWdlOiBuZ2lueAogICAgcG9ydHM6CiAgICAgIC0gdGFyZ2V0OiA4MAogICAgICAgIHB1Ymxpc2hlZDogIjgwIgo=}"
