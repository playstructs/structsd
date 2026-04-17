# structsd
**structsd** is the reference client for the Structs consensus network. 

This client is intended primarily for network validators and service providers. Most players within the Structs ecosystem will not need to operate the code in this repository. 

## Structs
In the distant future the species of the galaxy are embroiled in a race for Alpha Matter, the rare and dangerous substance that fuels galactic civilization. Players take command of Structs, a race of sentient machines, and must forge alliances, conquer enemies and expand their influence to control Alpha Matter and the fate of the galaxy.

[Structs](https://playstructs.com) is a decentralized game in the Cosmos ecosystem, operated and governed by our community of players--ensuring Structs remains online as long as there are players to play it.

## Get started
**structsd** is a [Cosmos SDK docs](https://docs.cosmos.network) project being developed with the help of the [Ignite CLI](https://ignite.com/cli). To learn more, see the [Ignite CLI docs](https://docs.ignite.com).

```
ignite chain serve
```

`serve` command installs dependencies, builds, initializes, and starts your blockchain in development.

## Using the Makefile

A `Makefile` is provided to wrap the most common build, test, lint, proto, dev, and release tasks. Run `make` (or `make help`) at the repo root to see every target with a short description.

Requires Go `1.23+`. Targets that touch protobuf, linting, or releases will pull their tools (`buf`, `golangci-lint`, `gofumpt`, `goreleaser`) on demand.

### Build

```
make build              # build ./build/structsd for the current platform
make install            # install structsd into $GOPATH/bin
make clean              # remove ./build artifacts
```

Cross-compile binaries (output in `./build/`):

```
make build-all                # all supported platforms
make build-linux-amd64
make build-linux-arm64
make build-darwin-amd64       # Intel Mac
make build-darwin-arm64       # Apple Silicon
make build-windows-amd64
```

Pass `LEDGER_ENABLED=false` to skip the Ledger build tag if you don't have `gcc` available, and `LINK_STATICALLY=true` for a fully static binary (Linux).

### Local development

These wrap the Ignite CLI and assume `ignite` is installed:

```
make serve                  # ignite chain serve
make serve-reset            # ignite chain serve --reset-once
make serve-reset-verbose    # ignite chain serve --reset-once --verbose
```

### Tests

```
make test                # go test ./...
make test-unit           # unit tests, 5m timeout, norace tag
make test-race           # tests with the race detector
make test-cover          # writes coverage.txt
make test-integration    # runs tests/test_chain.sh against a live chain
```

The integration script supports flags such as `--skip-mining` and `--extended-battle`; invoke it directly when you need them:

```
bash tests/test_chain.sh --skip-mining --extended-battle
```

### Linting & formatting

```
make lint        # golangci-lint
make lint-fix    # golangci-lint with --fix
make format      # gofumpt over the tree (skips generated .pb.go files)
```

### Protobuf

```
make proto-all       # format + lint + generate Go (gogo + pulsar) + TS
make proto-gen       # Go bindings only
make proto-gen-ts    # TypeScript bindings only
make proto-swagger   # OpenAPI/Swagger spec
make proto-format    # buf format -w
make proto-lint      # buf lint
```

### Release

```
make release-dry-run                          # local snapshot via goreleaser
GITHUB_TOKEN=<token> make release             # publish a release
```

In CI, releases are normally triggered by pushing a tag (`git tag v1.0.0 && git push origin v1.0.0`); the release workflow then runs `goreleaser` automatically.

### Misc

```
make go.sum    # go mod verify + tidy + download
make all       # build + lint + test
```

## Learn more

- [PlayStructs.com](https://playstructs.com)
- [Structs Wiki](https://watt.wiki)
- [@PlayStructs Twitter](https://twitter.com/playstructs)
- [/structs Farcaster Channel](https://warpcast.com/~/channel/structs)

## License

Copyright 2021 [Slow Ninja Inc](https://slow.ninja).

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.