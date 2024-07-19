# Exate APIGator Dora Router
This repository contains an HTTP proxy that receives an HTTP request from an
APIGator by Exate instance, forwards it to a list of APIGator instances, waits
for their responses, and filters the correct one to return it to the original
requester.

This behavior is designed to process information that may belong to other
countries and, depending on the laws and regulations of those countries, may be
shared or not under specific conditions. Every APIGator instance assigned to a
country must comply with the laws of the country in which it is configured.
Since the requester is not aware of every law, this proxy will take the request,
forward it to every APIGator instance, and retrieve the correct response if it
exists.

This component was designed for running on container environmnets
(K8s/Openshift) as a Stateless component.

## How it works
The APIGatorDoraRouter follows the next steps for every incoming request:
1. Loads the list of available APIGator instances and its properties from a INI
   config file
2. Receive HTTP request from the *Requester*.
3. Forwards the HTTP request to every configured APIGator.
    1. If there is no access token available for a specific APIGator instance,
       or it's expired, obtains a new one and continues.
    2. Sends the HTTP request to every APIGator instance (Multithreading)
    3. Waits for every APIGator response.
    4. Processes the responses looking for a correct one
    5. Based on configuration, this router will use different strategies for
       choosing the correct response. Check 
    6. If a correct response was found, returns its content as the response for
       the *requester* who started the process.


## Configuration
### Response Evaluation method
Currently there are two supported ways for choosing the "best" response on the
APIGatorDoraRouter:
1. First valid response. This method will return the first response with a
   correct data structure. Example:
   ```json
   {
     "dataSet": "{\n  \"employees\": {\n    \"employee\": […]\n  }\n}"
   }

   ```

   *To choose this method, edit the `config.ini` file on `[router].score_function='basic'*

2. DataSet with more fields decrypted. This method will choose the response
   based on which one has more decrypted information by APIGator. It takes the
   `restricted_text` field for identifying the crypted fields, and scores each
   response. The one with higher score (less crypted data) will be returned.

   *To choose this method, edit the `config.ini` file on `[router].score_function='percentage'*

## Running on Local
For an fast try on local, use the Makefile for starting the DoraRouter:
```sh
# Starts on normal mode
make start

# Starts on DEBUG mode for more verbose output
make start-debug
```

## Building
Every option for building and running this software is already defined on the
Makefile:
```sh
# Building container image
make build-image

# Pushing container image
make push
```

## Deployment on Openshift
The manifests for deploying the APIGator Dora Router on Openshift are available
on: `./manifests/deployment`
```sh
oc apply -f ./manifests/deployment
oc delete -f ./manifests/deployment
```

## Code Docs
For generating docs about the code, use the following command:
```sh
make docs
```

## Testing
There is a `scripts` folder on this repo which contains several scripts for
testing this component and its interaction with APIGator

Test script for APIGator:
```sh
# Generic command
bash ./scripts/test_gator.sh <APIGATOR_URL> <API_KEY> <CLIENT_ID> <CLIENT_SECRET> <PAYLOAD_FILE>
```

Test script for APIGatorDoraRouter:
```sh
# Generic command
bash ./scripts/test_router.sh <ROUTER_URL> <PAYLOAD_FILE>

# Example command
bash ./scripts/test_router.sh http://localhost:8080/forward ./tests/payload_example.json
```

## License
This software is released and distributed under the [Apache 2.0
License](./LICENSE).


## Maintainers:
* Alejandro Villegas (avillega@redhat.com)
