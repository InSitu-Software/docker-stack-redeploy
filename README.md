For the underlying problem see [this issue](https://github.com/moby/moby/issues/35048)

`stack-deploy` tries to implement the "suffix your config files" workaround using the docker packages for parsing and writing the compose.yaml file.

# Usage

    stack-redeploy --compose-file docker-compose.yml --stack stackname

options:

    -c, --compose-file string    Path to a Compose file (default "docker-compose.yml")
    -d, --docker-binary string   Alternative docker binary (default "docker")
    -o, --output                 Output YAML rather than redeploy
    -p, --prefix string          Prefix to be used for config prefix
    -s, --stack string           Stack to be redployed
      --with-registry-auth     Send registry authentication details to Swarm agents
    -w, --workdir string         Specify workdir (default ".")

# Implementation
stack-redeploy parses the given yaml using the [docker compose loader package](https://github.com/docker/cli/tree/master/cli/compose/loader). Afterwards it iterates over all top level config definitions and prefixes the config. Replacing all usages in services with the prefixed one. Finally it simply calls the `docker` binary to deploy the stack.

