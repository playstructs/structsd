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

Requires Go `1.27.3+`. Install `buf`, `goreleaser`, and `ignite` separately for the targets that use them. Lint and format targets install `golangci-lint` and `gofumpt` on demand; their binaries must be on your `PATH`.

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

#### Unicode and consensus

v0.22.0 requires Go `1.27.3` or newer for security fixes; release CI builds with Go `1.27.3`. Rebuild existing binaries to pick up the standard-library fixes. Operators using prebuilt binaries through Cosmovisor do not need Go installed on the node; the compiler requirement applies only to source builds. The character set the chain accepts for player, guild and planet names comes from Unicode 15.0.0 tables checked into `x/structs/types/unicode_tables.go`, not from the Unicode tables of the toolchain that compiled the binary, so changing the build toolchain does not change these name rules.

`golang.org/x/text` remains pinned to `v0.28.0` through a module replacement, even when networking dependencies request a newer version. Changing that replacement requires a separate consensus review and upgrade handler.

This matters because Go resolves `\p{L}` in a regexp, and `unicode.Is` against `unicode.L`, from the compiling toolchain's standard library, and those tables grow with Go releases. `U+088F` is unassigned in Unicode 15.0.0 and a letter in later versions, so before v0.21.0 a name built from it was accepted by validators on a newer Go and rejected by the rest — with only the accepting side writing the name, which is an application state split. The tables are now state.

```
make check_unicode      # does this toolchain still ship the pinned Unicode version?
```

The answer does not change whether your binary is consensus-compatible; it only tells you whether `go test` can still verify the checked-in tables against the standard library. On a toolchain past Unicode 15.0.0 that comparison skips and the tables are covered by a checksum and by fixed cross-script vectors instead. **Do not run `make unicode-tables` to resolve a mismatch** — regenerating the tables changes which names the chain accepts and requires an upgrade handler.

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
make test-unit           # same Go packages, 5m timeout, norace tag
make test-race           # tests with the race detector
make test-cover          # writes coverage.txt
make test-integration    # runs tests/test_chain.sh against a live chain
```

Go package tests include the in-process app and security regression tests. Live-chain tests are separate and require a disposable development chain. To limit hashing and run extended battles against an already running development chain:

```
make test-integration TEST_CHAIN_FLAGS="--skip-mining --extended-battle"
```

For an isolated lifecycle run with frozen source/binary evidence, export, and restart checks, use the Python 3 runner:

```
make build
python3 -B tests/run_full_lifecycle.py --binary build/structsd --output-root build --skip-mining
```

The runner always enables extended battles and battle logging. `--skip-mining` skips mining/refining but still computes build proofs. Stop other local Structs nodes and any containers occupying the chain's ports before starting; the runner does not stop unrelated processes. Evidence is retained under `build/full-lifecycle-*`.

See the [v0.22.0 vulnerability review](docs/upgrades/v0.22.0-vulnerability-review.md) for dependency fixes, the retained `x/text` pin, and final lifecycle/Cosmovisor validation results. Tests must be run locally; the release workflow does not run them.

### Linting & formatting

```
make lint        # golangci-lint
make lint-fix    # golangci-lint with --fix
make format      # gofumpt over the tree (skips generated .pb.go files)
```

The repository has an existing lint backlog. For focused changes, scope linting to the affected packages and avoid tree-wide formatting, which rewrites unrelated files.

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
- [Structs Codex](https://playstructs.com/codex)
- [Structs Developers Portal](https://watt.wiki)
- [Structs Agentic Documentation](https://structs.ai/llms.txt)
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
