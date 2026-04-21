package http

import "strconv"

// strconvItoa is a tiny indirection because the test file avoids pulling
// the strconv import into the rest of the package. Kept in a separate
// *_test.go file so it doesn't ship in non-test builds.
func strconvItoa(i int) string { return strconv.Itoa(i) }
