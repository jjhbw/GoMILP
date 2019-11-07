# WARNING
<u>*While this project has taught me a lot, I have stopped working on it for now due to it being a significant time sink. Solvers are complex! Who would've thought? ;)
In its current state it only solves the most trivial of MILP problems.*</u>

# GoMILP

The scope of this project is to build a simple *Mixed Integer Linear Program* (MILP) solver with an easy to use API in pure Go. Several alternatives ([1](https://github.com/draffensperger/golp),[2](https://github.com/lukpank/go-glpk),[3](https://github.com/costela/golpa)) exist in the form of CGO bindings with older LP solver libraries. While excellent pieces of software, I found their dependence on external libraries a big downside for use cases where maximum portability is key.

This project features an implementation of a ('lazy') branch-and-bound method for solving Mixed Integer Linear Programs. The applied branch and bound procedure is basically a heuristic-guided  search over an enumeration tree that is generated on the fly. In this tree, each node is a particular relaxation of the original problem with additional heuristically determined constraints. To solve each LP relaxation, we use [Gonum's excellent implementation]() of the [Simplex](https://en.wikipedia.org/wiki/Simplex_algorithm) algorithm.



# Dependencies

Go dependencies are managed using [Go dep](https://github.com/golang/dep). See `Gopkg.lock` and `Gopkg.toml`.

For testing, solutions to randomized MILPs are compared to solutions produced by the *GNU Linear Programming Kit*, using its [Go bindings](https://github.com/lukpank/go-glpk). To install `libglpk` on macOs, simply run `brew install glpk`.



# TODO

### Hurdles

- [ ] Prevent matrix singularity introduced by problem preprocessing 
- [ ] Problem preprocessing code is still messy and needs some unit tests.
- [ ] All rummikub problems in the unit tests of `rummiGo` have integer-feasible initial relaxations…  Write tests with more complex problems that tax the branch-and-bound procedure
- [ ] Lack of a simple presolver precludes tackling larger/more complex problems (see the Huang thesis in references). See `presolver` branch.
  - [x] removing empty rows (all zeroes)
  - [ ] removing empty columns
  - [x] removing (implicitly) fixed variables
    - currently modifies the original problem (through a variable pointer), this is ugly!
- [x] remove duplicated rows
  - [ ] substitute singleton rows
  - [ ] substitute singleton columns


### Enhancements

- [ ] extend the instrumentation hook to allow instrumentation (logging) of the presolver operations
- [ ] primal/dual solving
- [ ] problem preprocessing of non-root nodes in the enumeration tree?
- [ ] variables are currently subject to nonnegativity constraints by default.
- [ ] Formal testing against [problems with known solutions](http://miplib.zib.de/miplib2010.php)? ([MPS parser](https://github.com/dennisfrancis/mps) needed)
- [ ] Cancellation currently only possible when bnb procedure has been started. We may want to be able to cancel the solving of the initial relaxation too.
- [ ] Deal with infeasible subproblems created after branching on a particular integrality-constrained variable of a LP feasible problem. Should this be a noop (currently) or should branching be retried on another integer constrained variable?
- [ ] Enumeration tree exploration queue is currently FIFO. For depth-first exploration, we should go with a LIFO queue.
- [ ] Add heuristic determining which node gets explored first (as we are using depth-first search) https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html?s_tid=gn_loc_drop#btzwtmv
- [ ] how to deal with matrix degeneracy in subproblems? Currently handled the same way as infeasible subproblems.
- [ ] In branched subproblems: is it sensible to intiate the simplex at solution of parent? (using argument of lp.Simplex)
- [ ] does fiddling with the simplex tolerance value improve outcomes?
- [ ] Currently implemented only the simplest branching heuristics. Room for improvement such as expensive branching heuristics like node (pseudo-)costs.
- [ ] Enumeration tree exploration heuristics: use priority queue-based on heuristics like total path cost or a best-first approach based on earlier solutions.


- [ ] CI procedure should include race detector and test timeouts
- [ ] sanity checks before converting Problem to a MILPproblem, such as NaN, Inf, and matrix shapes and variable bound domains.
- [ ] write benchmarks for time (and space?) usage
- [ ] small(?) performance gains may be made by switching dense matrix datastructures over to sparse ones for bigger problems. This could be facilitated by employing Gonum's mat.Matrix interface.
