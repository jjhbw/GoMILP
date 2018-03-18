# Simple MILP solver

A work in progress. Currently features an implementation of (lazy) branch-and-bound for solving Mixed Integer Linear Programs, using Gonum's Simplex algorithm to solve each LP relaxation in the tree.

**Currently only supports basic problem formulation:**

- [x] integrality constraints
- [x] inequality constraints
- [x] equality constraints
- [x] maximization
- [x] minimization
- [ ] Optional relaxation of nonnegativity constraints (allow decision variables to take negative values)

**Cool features for future iterations:**

- [ ] Problem preprocessing before solving (e.g. removing redundant constraints)
- [ ] Parallelism of the branch-and-bound procedure

# Scope

MILP solver with a simple interface. Speed is not yet a concern. 

# Dependencies

Go dependencies are managed using [Go dep](https://github.com/golang/dep). See `Gopkg.lock` and `Gopkg.toml`.

For testing, solutions to randomized MILPs are compared to solutions produced by the GNU Linear Programming Kit, using its [Go bindings](https://github.com/lukpank/go-glpk). To install `libglpk` on macOs, simply run `brew install glpk`.



