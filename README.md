# load-balancers

`load-balancers` provides in-process, weighted selection algorithms for Go.
Each package selects a registered component ID; transport, health checking,
retries, and connection management remain the responsibility of the caller.

Use it when an application needs to choose between known backends—for example,
HTTP upstreams, worker queues, or database replicas—and already owns the
operational policy around those backends.

## Packages

| Package | What it does | Best when |
| --- | --- | --- |
| [`weightedrandom`](./weightedrandom) | Selects a component randomly in proportion to its weight. | Long-run distribution matters more than sequence order. |
| [`weightedroundrobin`](./weightedroundrobin) | Cycles through components in exact weighted proportions. | You need deterministic allocation over a complete cycle. |

Weights must be positive integers. A component with weight `6` receives six
times as many selections as one with weight `1`.

## Install

```sh
go get github.com/the-code-genin/load-balancers
```

A minimum go version of `1.25` is required.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/the-code-genin/load-balancers/weightedrandom"
)

func main() {
	lb := weightedrandom.NewLoadBalancer()

	if err := lb.Register("primary", 3); err != nil {
		log.Fatal(err)
	}
	if err := lb.Register("replica", 1); err != nil {
		log.Fatal(err)
	}

	id, err := lb.Select()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("send the request to", id)
}
```

The round-robin package has the same `NewLoadBalancer`, `Register`,
`Unregister`, and `Select` API:

```go
lb := weightedroundrobin.NewLoadBalancer()
_ = lb.Register("api-a", 2)
_ = lb.Register("api-b", 1)

// One cycle contains two selections of api-a and one of api-b.
id, err := lb.Select()
```

`Register` rejects non-positive weights. `Select` returns an error when no
components are registered, and `Unregister` returns an error for an unknown
component.

## How the algorithms behave

### Weighted random

`weightedrandom` maps each component to a portion of a selection range based
on its weight, then samples that range with `crypto/rand`. The result is
probabilistic: over a sufficiently large number of selections, the observed
distribution approaches the configured weight ratio. It does not guarantee
fairness over a short interval.

### Weighted round robin

`weightedroundrobin` assigns each component a contiguous section of a cycle
and advances one position on each call. A complete cycle contains each
component exactly as many times as its weight. Updating membership or a weight
rebuilds and restarts that cycle.

This is **weighted cyclic round robin**, not smooth weighted round robin.
Selections for a higher-weight component can be adjacent rather than evenly
spread through the cycle. If even spacing matters for latency or burst control,
use a smooth-WRR implementation instead.

The cycle is ordered by ascending weight, then ID, making its sequence
repeatable for a given set of components.

## Concurrency and operational scope

Both packages are safe for concurrent calls to `Register`, `Unregister`, and
`Select`. A membership or weight update is serialized with selection, so a
selection observes either the old configuration or the rebuilt one.

These packages intentionally do not manage backend health, draining, retries,
connection pools, or metrics. Integrate those concerns in the caller, and
register, update, or unregister a component as its state changes.

### Limitations

- Both implementations perform a linear scan of registered components during
  selection. They are designed for small to moderate backend sets, not very
  large routing tables.
- Total weight is stored in a Go `int`. Extremely large aggregate weights can
  overflow and are not supported; validate externally supplied weights before
  registration.
- `weightedroundrobin` is weighted cyclic round robin, not smooth weighted
  round robin. A high-weight component may receive adjacent selections.

## Development

Run the project test target before opening a pull request:

```sh
make test
```

It runs the complete suite verbosely with the race detector and a five-minute
timeout.

## Contributing

Issues and pull requests are welcome. Keep changes focused, add tests for
behavior changes, and run the commands above before opening a pull request.
For an API or algorithm change, open an issue first so the expected semantics
can be agreed before implementation.

## Maintainer

Maintained by [Mohammed Adekunle](https://github.com/the-code-genin). Use the
[issue tracker](https://github.com/the-code-genin/load-balancers/issues) for
bug reports, questions, and feature proposals.

## License

Distributed under the [Apache License 2.0](./LICENSE).
