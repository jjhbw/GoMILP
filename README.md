# Simple MILP solver (no CGO)

A work in progress.

# Scope

MILP solver with a simple interface. Speed is not yet a concern.

# Dependencies

Go dependencies are managed using [Go dep](https://github.com/golang/dep). See `Gopkg.lock` and `Gopkg.toml`.

For testing, solutions to randomized MILPs are compared to solutions produced by the GNU Linear Programming Kit, using its [Go bindings](https://github.com/lukpank/go-glpk).

To install `libglpk` on macOs, simply run `brew install glpk`.



