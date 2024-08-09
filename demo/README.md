# Demo script for APIGatorDoraRouter
This folder contains a script for testing the APIGatorDoraRouter

## Testing
Before running the demo, copy the `config.env.example` file and tune it with
the values for your run.
```sh
cp config.env.example config.env
vim config.env
```
When configuring your demo, consider that this script needs to obtain a APIGator
Token before start, and it will use the DECODER properties for it.

For running the test, use the following command:
```sh
bash demo.sh config.env
```
