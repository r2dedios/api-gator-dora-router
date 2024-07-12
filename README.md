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
    5. If a correct response was found, returns its content as the response for
       the *requester* who started the process.


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

## License
This software is released and distributed under the [Apache 2.0
License](./LICENSE).


## Maintainers:
* Alejandro Villegas (avillega@redhat.com)
