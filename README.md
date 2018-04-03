# GoMILP

<u>*A work in progress.*</u>

The scope of this project is to build a simple, reliable MILP solver with an easy to use API in pure Go. Several alternatives ([1](https://github.com/draffensperger/golp),[2](https://github.com/lukpank/go-glpk),[3](https://github.com/costela/golpa)) exist in the form of CGO bindings with older LP solver libraries. While excellent pieces of software, I found their dependence on external libraries a big downside for usecases where maximum portability is key.

This project features an implementation of a ('lazy') branch-and-bound method for solving Mixed Integer Linear Programs. The applied branch and bound procedure is basically a heuristic-guided (depth-first **?**) search over an enumeration tree that is generated on the fly. In this tree, each node is a particular relaxation of the original problem with additional heuristically determined constraints. To solve each LP relaxation, we use [Gonum's excellent implementation]() of the [Simplex](https://en.wikipedia.org/wiki/Simplex_algorithm) algorithm.



# Dependencies

Go dependencies are managed using [Go dep](https://github.com/golang/dep). See `Gopkg.lock` and `Gopkg.toml`.

For testing, solutions to randomized MILPs are compared to solutions produced by the GNU Linear Programming Kit, using its [Go bindings](https://github.com/lukpank/go-glpk). To install `libglpk` on macOs, simply run `brew install glpk`.



# TODO

### Hurdles

- [ ] Problem preprocessing code is still messy and fragile. Remove/replace presolve operations that work on the problem matrix instead of on the problem's abstract definition?
- [ ] Only the simples problem preprocessing steps have been implemented.
- [ ] Problem preprocessing: matrices of e.g. rummikub problems can be greatly simplified by removing redundant constraints.
- [ ] Debug ostensibly simple rummikub sub-problems that take a very long time to solve.  **Hypothesis**: simplex panics like `lp: bland: all replacements are negative or cause ill-conditioned ab` can be prevented using aggressive problem preprocessing.
- [ ] Branching procedure may generate new constraints that are superseded by existing ones (e.g. branching on `x1 >= 2 | X <= 1` when there is already an existing constraint stating that `x1 < 1` ). This is wasteful and can be solved by more intelligent branching.
- [ ] Somehow all rummikub problems in the unit tests of rummiGo have integer-feasible initial relaxations… The resulting solutions do perfectly match the specified test results, though.


### Enhancements

- [ ] primal/dual solving
- [ ] problem preprocessing of non-root nodes in bnb tree
- [ ] variables are currently subject to negativity constraints by default.
- [ ] Formal testing against [problems with known solutions](http://miplib.zib.de/miplib2010.php)? ([MPS parser](https://github.com/dennisfrancis/mps) needed)
- [ ] Convert subproblem to standard form in an earlier stage (remove inequality matrix asap). Lots of room for optimization in the `combineInequalities` and `convertToEqualities` functions.
- [ ] Cancellation currently only possible when bnb procedure has been started. We may want to be able to cancel the solving of the initial relaxation too.
- [ ] Deal with infeasible subproblems created after branching on a particular integrality-constrained variable of an LP feasible problem. Should this be a noop (currently) or should branching be retried on another integer constrained variable?
- [ ] Queue is currently FIFO. For depth-first exploration, we should go with a LIFO queue.
- [ ] Add heuristic determining which node gets explored first (as we are using depth-first search) https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html?s_tid=gn_loc_drop#btzwtmv
- [ ] Maybe make context.Context optional only in the top-level API
- [ ] Low-level solver cancellation plumbing may be better off governed by our own cancellation hooks instead of the bulky context API. On the other hand, context has a nice set of errors defined on deadline exceeded etc. Replacing it may not be worth it.
- [ ] how to deal with matrix degeneracy in subproblems? Currently handled the same way as infeasible subproblems.
- [ ] in branched subproblems: intiate simplex at solution of parent? (using argument of lp.Simplex)
- [ ] does fiddling with the simplex tolerance value improve outcomes?
- [ ] Currently implemented only the simplest branching heuristics. Room for improvement such as expensive branching heuristics like node (pseudo-)costs.
- [ ] Enumeration tree exploration heuristics: use priority queue based on heuristics like total path cost or a best-first approach based on earlier solutions.
- [ ] also fun: linear program preprocessing (MATLAB docs: https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html#btv20av). This may shave off time and space complexity when applying the Simplex solver on the subproblems.


- [ ] CI procedure should include race detector and test timeouts
- [ ] sanity checks before converting Problem to a MILPproblem, such as NaN, Inf, and matrix shapes and variable bound domains
- [ ] testing against GLPK is extremely convoluted due to its shitty API. Moreover, its output is sometimes plain wrong (doesnt diagnose unbounded problems).
- [ ] try to formulate more advanced constraints, like sets of values instead of just integrality? Note that having integer sets as constraints is basically the same as having an integrality constraint, and a <= and >= bound. Branching on this type of constraint can be optimized in a neat way (i.e.) ` x>=0, x<=1, x<=0 ~-> x = 0)`
- [ ] dealing with variables that are unrestricted in sign (currently, each var is subject to a nonnegativity constraint)
- [ ] make CLI and Problem serialization format for easy integration with R/python-based analysis tooling for debugging of mathematical properties.
- [ ] write benchmarks for time and space usage
- [ ] small(?) performance gains may be made by switching dense matrix datastructures over to sparse ones for bigger problems. This could be facilitated by employing Gonum's mat.Matrix interface.