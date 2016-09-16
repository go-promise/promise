/**********************************************************\
 *                                                        *
 * promise/state.go                                       *
 *                                                        *
 * promise state for Go.                                  *
 *                                                        *
 * LastModified: Aug 13, 2016                             *
 * Author: Ma Bingyao <andot@hprose.com>                  *
 *                                                        *
\**********************************************************/

package promise

// State of the promise.
type State int

// Promise state enum values.
const (
	PENDING = State(iota)
	FULFILLED
	REJECTED
)
