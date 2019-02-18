package httpserve

import (
	"testing"
)

func TestRouteCheck(t *testing.T) {
	var (
		r   *route
		err error
	)

	if r, err = newRoute(smallRoute, nil, "GET"); err != nil {
		t.Fatal(err)
	}

	params, ok := r.check(nil, smallRouteNoParam)
	if !ok {
		t.Fatal("Match not ok when it should be")
	}

	if value := params.ByName("name"); value != "name" {
		t.Fatalf("Invalid value for key \"%s\", expected \"%s\" and received \"%s\"", "name", "name", value)
	}

	params, ok = r.check(params, "test")
	if ok {
		t.Fatal("Match ok when it should not be")
	}
}
