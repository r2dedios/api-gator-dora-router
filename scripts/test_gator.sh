[[ $# != 5 ]] && { 
  echo "Missing arguments. Run example: 'bash ./scripts/test_gator <APIGATOR_URL> <API_KEY> <CLIENT_ID> <CLIENT_SECRET> <PAYLOAD_FILE>'"
  exit
}

APIGATOR_URL=$1
API_KEY=$2
CLIENT_ID=$3
CLIENT_SECRET=$4
PAYLOAD_FILE=$5


echo "Obtainning access Token from $APIGATOR_URL"
token=$(curl --silent --location "$APIGATOR_URL/apigator/identity/v1/token" \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --header "X-Api-Key: $API_KEY" \
  --data-urlencode "client_id=$CLIENT_ID" \
  --data-urlencode "client_secret=$CLIENT_SECRET" \
  --data-urlencode 'grant_type=client_credentials' | jq -r '.access_token')


echo "Token obtained. Sending test payload"
curl --location "$APIGATOR_URL/apigator/protect/v1/dataset" \
  --header "X-Api-Key: $API_KEY" \
  --header 'X-Data-Set-Type: JSON' \
  --header "X-Resource-Token: Bearer $token" \
  --header 'Content-Type: application/json' \
  -w "%{http_code}" \
  -d@$PAYLOAD_FILE | jq '.dataSet | fromjson'

