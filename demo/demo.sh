#!/bin/bash
# This script is designed for ilustrating on a live demo how the APIGatorDoraRouter developed by RedHat in collaboration with Exate works
# Exate APIGator: https://www.exate.com/apigator
# APIGatorDoraRouter repo: https://github.com/RHEcosystemAppEng/api-gator-dora-router
# Author: Alejandro Villegas López <avillega@redhat.com>
################################################################################


# Global vars:
################################################################################
OK=0
ERR=-1

# checking args num
[[ $# -ne 1 ]] && { echo "Wrong number of arguments. Expeceted run: $0 <CONFIG_FILE_PATH>"; exit; }

# checking config file introduced by arguments
CONFIG_FILE=$1
[[ ! -f $CONFIG_FILE ]] && { echo "The config file indicated by args ($CONFIG_FILE) doesn't exist"; exit; } || { echo "Loading $CONFIG_FILE"; source $CONFIG_FILE; }

# Functions
################################################################################

# Standard info message function
function echo_inf () {
  echo -e "[\033[34mINF\033[0m]: $@"
}

# Standard debug message function
function echo_debug () {
  if [[ $VERBOSE -eq 1 ]]; then
    echo -e "[\033[33mDEBUG\033[0m]: $@"
  fi
}

# Standard error message function
function echo_err () {
  echo -e "[\033[31mERR\033[0m]: $@"
}

# validate_file_exists checks if the file introduced by arguments exists or not
function validate_file_exists () {
  local file=$1

  if [[ -f $file ]]; then
    return $OK
  else
    return $ERR
  fi
}

# validate_url evaluates the received argument to determine if its content fits with an URL structure
function validate_url () {
  local url=$1
  regex='(https?|ftp|file)://[-[:alnum:]\+&@#/%?=~_|!:,.;]+'

  if [[ $url =~ $regex ]]; then
    return $OK
  else
    return $ERR
  fi
}

# validate_non_empty checks if the incoming argument is emtpy or not by returning $ERR or $OK
function validate_non_empty () {
  local val=$1

  if [[ -z "$val" ]]; then
    return $ERR
  else
    return $OK
  fi
}

# validate_decoder_properties checks every config parameter required by this script about the decoder APIGator instance
function validate_decoder_properties () {
  # Check decoder API URL
  validate_url $APIGATOR_DECODER_API_URL
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_DECODER_API_URL: ($APIGATOR_DECODER_API_URL) Is not a valid URL"
    return $ERR
  fi

  # Validating rest of parameters are not empty
  validate_non_empty $APIGATOR_DECODER_API_KEY
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_DECODER_API_KEY is empty"
    return $ERR
  fi

  validate_non_empty $APIGATOR_DECODER_CLIENT_ID
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_DECODER_CLIENT_ID is empty"
    return $ERR
  fi

  validate_non_empty $APIGATOR_DECODER_CLIENT_SECRET
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_DECODER_CLIENT_SECRET is empty"
    return $ERR
  fi
}

# validate_encoder_properties checks every config parameter required by this script about the encoder APIGator instance
function validate_encoder_properties () {
  # Check encoder API URL
  validate_url $APIGATOR_ENCODER_API_URL
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_API_URL: ($APIGATOR_ENCODER_API_URL) Is not a valid URL"
    return $ERR
  fi

  # Validating rest of parameters are not empty
  validate_non_empty $APIGATOR_ENCODER_API_KEY
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_API_KEY is empty"
    return $ERR
  fi

  validate_non_empty $APIGATOR_ENCODER_CLIENT_ID
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_CLIENT_ID is empty"
    return $ERR
  fi

  validate_non_empty $APIGATOR_ENCODER_CLIENT_SECRET
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_CLIENT_SECRET is empty"
    return $ERR
  fi

  validate_file_exists $APIGATOR_ENCODER_PAYLOAD_FILE
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_PAYLOAD_FILE: ($APIGATOR_ENCODER_PAYLOAD_FILE) Doesn't exist"
    return $ERR
  fi

  validate_file_exists $APIGATOR_ENCODER_PAYLOAD_EXPECTED_FILE
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_PAYLOAD_EXPECTED_FILE: ($APIGATOR_ENCODER_PAYLOAD_FILE) Doesn't exist"
    return $ERR
  fi
}

# validate_dora_router_properties checks every config parameter required by this script about the APIGatorDoraRouter instance
function validate_dora_router_properties () {
  # Check encoder API URL
  validate_url $APIGATOR_ENCODER_API_URL
  if [[ $? != $OK ]]; then
    echo_err "APIGATOR_ENCODER_API_URL: ($APIGATOR_ENCODER_API_URL) Is not a valid URL"
    return $ERR
  fi
}


# login_apigator performs the login operation againts the APIGator instance and returns the obtained Barear Token or error.
# The token can be resued for encoder and decoder. APIGatorDoraRouter will take its own token
function login_apigator () {
  response="$(curl \
    -X POST --silent \
    --location "$APIGATOR_DECODER_API_URL/apigator/identity/v1/token" \
    --header "X-Api-Key: $APIGATOR_DECODER_API_KEY" \
    --header 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode "client_id=$APIGATOR_DECODER_CLIENT_ID" \
    --data-urlencode "client_secret=$APIGATOR_DECODER_CLIENT_SECRET" \
    --data-urlencode 'grant_type=client_credentials'
  )"

	validate_non_empty "$response"
  if [[ $? != $OK ]]; then
    echo_err "Token response is empty. Check the login curl command"
    return $ERR
  fi

  echo "$response"  | jq '.access_token'
}

function encoder_crypt () {
  result=$(curl \
    -X GET --silent \
    --location "$APIGATOR_ENCODER_API_URL/rp/v1/en/v3/objects/contacts" \
    --header "Content-Type: application/json" \
    --header "X-Api-Key: $APIGATOR_ENCODER_API_KEY" \
    --header "X-ADGroup: Marketing" \
    --header "X-UsageType: 211" \
    --header "X-JobType: Pseudonymise" \
    --header "X-SnapShotDate: 2023-03-21T18:56:24Z" \
    --header "X-ClientId: $APIGATOR_ENCODER_CLIENT_ID" \
    --header "X-ClientSecret: $APIGATOR_ENCODER_CLIENT_SECRET" \
    -d@$APIGATOR_ENCODER_PAYLOAD_FILE
  )

  echo "$result"
}

# parse_data prepares the payload to be sent to an APIGator instance
function parse_data () {
  content=$(echo "$1" | jq -c '.' | tr '"' "'" )
	jq --arg content "$content" '.dataSet = $content' <<< "$APIGATOR_DORA_ROUTER_RECONSTRUCT_PAYLOAD" | sed 's/\\n//g'
}

# forward_dora_router sends the parsed payload to the APIGatorDoraRouter
function forward_dora_router () {
  curl \
    -X POST --silent \
    --insecure \
    --location "$APIGATOR_DORA_ROUTER_URL/forward" \
    -d@"$1" | jq '.dataSet | fromjson' 2>/dev/null
}

# Main
function main () {
  # Validation Stage
  ############################################################
	echo_inf "Reading configuration"
  validate_decoder_properties
  if [[ $? != $OK ]]; then
    echo_err "Configuration for the Encoder is wrong. Review your config.env file"
    return $ERR
  fi
  validate_encoder_properties
  if [[ $? != $OK ]]; then
    echo_err "Configuration for the Encoder is wrong. Review your config.env file"
    return $ERR
  fi
  validate_dora_router_properties
  if [[ $? != $OK ]]; then
    echo_err "Configuration for the APIGatorDoraRouter is wrong. Review your config.env file"
    return $ERR
  fi

  # Login Stage
  ############################################################
	# Using the DECODER properties for login and obtain a token
  echo_inf "Obtainning token"
  token=$(login_apigator $APIGATOR_DECODER_API_URL $APIGATOR_DECODER_API_KEY $APIGATOR_DECODER_CLIENT_ID $APIGATOR_DECODER_CLIENT_SECRET)
  if [[ $? != $OK ]]; then
    echo_err "Login Error. Aborting"
    return $ERR
  fi
  echo_debug "Token Obtained!: $(echo $token | md5sum)"


  # Encoding stage
  ############################################################
	echo_inf "Encoding payload with Encoder"
  crypted_result=$(encoder_crypt $APIGATOR_ENCODER_API_URL $APIGATOR_ENCODER_API_KEY $token $APIGATOR_ENCODER_CLIENT_ID $APIGATOR_ENCODER_CLIENT_SECRET $APIGATOR_ENCODER_PAYLOAD_FILE)
  if [[ $? != $OK ]]; then
    echo_err "Failed to encode the payload on the Encoder"
    return $ERR
  fi
	echo_debug "Response from Encoder: $(echo $crypted_result)"

	echo_inf "Parsing data before forwarding"
  parsed_dataSet=$(parse_data "$crypted_result")
  echo "$parsed_dataSet" > ./dataset.json

  # Forwarding to Router Stage
  ############################################################
  echo_inf "Forwarding encoded result to APIGatorDoraRouter"
  router_response="$(forward_dora_router ./dataset.json)"

  echo_debug "Router Response: \n$(echo $router_response)"
  echo_debug "Expected Response:\n$(cat $APIGATOR_ENCODER_PAYLOAD_FILE | jq)"

  # Comparing results Stage
  ############################################################
  echo_inf "Evaluating results:"

  echo_inf "Diff between encoded dataSet and Router response"
  echo_inf "\033[33mEncoded Dataset:                                       |  Router Response:\033[0m"
  diff --color --side-by-side <(echo $crypted_result | jq) <(echo "$router_response") | colordiff

	echo "$router_response" > router_response.json
	hash_router_response=($(md5sum router_response.json))
	hash_expected_response=($(md5sum $APIGATOR_ENCODER_PAYLOAD_EXPECTED_FILE))
  echo_inf "Router response data:   $hash_router_response"
  echo_inf "Expected response data: $hash_expected_response"
	if [[ "$hash_router_response" == "$hash_expected_response" ]]; then
		echo_inf "Result correct!"
	fi

}

echo_inf "Starting Demo"
time main
echo_inf "Done!"
