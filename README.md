# okta-cli
A golang binary to run plays against the okta api, similiar to how ansible works.

It reads a yaml file, and creates a new user, or makes an existing user match what is in the playbook.

## Examples
```/bin/bash

$ export OKTAURL=https://example.okta.com # Replace with Okta you want to hit
$ export OKTATOKEN=00BwpAe_ZczokMGhKIvX # Replace with token to use with above
$ okta-cli --playbook /opt/playbooks/jimmy.yaml
```
