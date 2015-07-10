package execution

import (
	"pfi/sensorbee/sensorbee/bql/udf"
	"pfi/sensorbee/sensorbee/core"
	"pfi/sensorbee/sensorbee/data"
)

type defaultSelectExecutionPlan struct {
	streamRelationStreamExecutionPlan
}

// CanBuildDefaultSelectExecutionPlan checks whether the given statement
// allows to use an defaultSelectExecutionPlan.
func CanBuildDefaultSelectExecutionPlan(lp *LogicalPlan, reg udf.FunctionRegistry) bool {
	return !lp.GroupingStmt
}

// defaultSelectExecutionPlan is a very simple plan that follows the
// theoretical processing model. It does not support aggregration.
//
// After each tuple arrives,
// - compute the contents of the current window using the
//   specified window size/type,
// - perform a SELECT query on that data,
// - compute the data that need to be emitted by comparison with
//   the previous run's results.
func NewDefaultSelectExecutionPlan(lp *LogicalPlan, reg udf.FunctionRegistry) (ExecutionPlan, error) {
	underlying, err := newStreamRelationStreamExecutionPlan(lp, reg)
	if err != nil {
		return nil, err
	}
	return &defaultSelectExecutionPlan{
		*underlying,
	}, nil
}

// Process takes an input tuple and returns a slice of Map values that
// correspond to the results of the query represented by this execution
// plan. Note that the order of items in the returned slice is undefined
// and cannot be relied on.
func (ep *defaultSelectExecutionPlan) Process(input *core.Tuple) ([]data.Map, error) {
	return ep.process(input, ep.performQueryOnBuffer)
}

// performQueryOnBuffer executes a SELECT query on the data of the tuples
// currently stored in the buffer. The query results (which is a set of
// data.Value, not core.Tuple) is stored in ep.curResults. The data
// that was stored in ep.curResults before this method was called is
// moved to ep.prevResults. Note that the order of values in ep.curResults
// is undefined.
//
// In case of an error the contents of ep.curResults will still be
// the same as before the call (so that the next run performs as
// if no error had happened), but the contents of ep.curResults are
// undefined.
//
// Currently performQueryOnBuffer can only perform SELECT ... WHERE ...
// queries without aggregate functions, GROUP BY, or HAVING clauses.
func (ep *defaultSelectExecutionPlan) performQueryOnBuffer() error {
	// reuse the allocated memory
	output := ep.prevResults[0:0]
	// remember the previous results
	ep.prevResults = ep.curResults

	rollback := func() {
		// NB. ep.prevResults currently points to an slice with
		//     results from the previous run. ep.curResults points
		//     to the same slice. output points to a different slice
		//     with a different underlying array.
		//     in the next run, output will be reusing the underlying
		//     storage of the current ep.prevResults to hold results.
		//     therefore when we leave this function we must make
		//     sure that ep.prevResults and ep.curResults have
		//     different underlying arrays or ISTREAM/DSTREAM will
		//     return wrong results.
		ep.prevResults = output
	}

	// we need to make a cross product of the data in all buffers,
	// combine it to get an input like
	//  {"streamA": {data}, "streamB": {data}, "streamC": {data}}
	// and then run filter/projections on each of this items

	dataHolder := data.Map{}

	// function to evaluate filter on the input data and -- if the filter does
	// not exist or evaluates to true -- compute the projections and store
	// the result in the `output` slice
	evalItem := func(d data.Map) error {
		// evaluate filter condition and convert to bool
		if ep.filter != nil {
			filterResult, err := ep.filter.Eval(d)
			if err != nil {
				return err
			}
			filterResultBool, err := data.ToBool(filterResult)
			if err != nil {
				return err
			}
			// if it evaluated to false, do not further process this tuple
			// (ToBool also evalutes the NULL value to false, so we don't
			// need to treat this specially)
			if !filterResultBool {
				return nil
			}
		}
		// otherwise, compute all the expressions
		result := data.Map(make(map[string]data.Value, len(ep.projections)))
		for _, proj := range ep.projections {
			value, err := proj.evaluator.Eval(d)
			if err != nil {
				return err
			}
			if err := assignOutputValue(result, proj.alias, value); err != nil {
				return err
			}
		}
		output = append(output, result)
		return nil
	}

	// Note: `ep.buffers` is a map, so iterating over its keys may yield
	// different results in every run of the program. We cannot expect
	// a consistent order in which evalItem is run on the items of the
	// cartesian product.
	allStreams := make([]string, 0, len(ep.buffers))
	for key := range ep.buffers {
		allStreams = append(allStreams, key)
	}
	if err := ep.processCartesianProduct(dataHolder, allStreams, evalItem); err != nil {
		rollback()
		return err
	}

	ep.curResults = output
	return nil
}
