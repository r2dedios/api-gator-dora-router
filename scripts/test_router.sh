[[ $# != 2 ]] && { echo "Missing arguments. Run example: 'bash ./scripts/test_router.sh <ROUTER_URL> <PAYLOAD_FILE>'"; exit; }

ROUTER_URL=$1
PAYLOAD_FILE=$2

echo "Testing APIGator Dora Router running on $ROUTER_URL using $PAYLOAD_FILE as Payload"

curl --silent --location $ROUTER_URL \
  --insecure \
  -w "HTTP RC: %{http_code}" \
  -d@$PAYLOAD_FILE | jq '.dataSet | fromjson'
