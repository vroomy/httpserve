package httpserve

import (
	"testing"
)

func TestRouteCheck(t *testing.T) {
	r := newRoute(smallRoute, nil, methodGET)
	params := Params{}
	ok := r.check(smallRouteNoParam, params)
	if !ok {
		t.Fatal("Match not ok when it should be")
	}

	if key, value := "name", params["name"]; value != "name" {
		t.Fatalf("Invalid value for key \"%s\", expected \"%s\" and received \"%s\"", key, key, value)
	}

	ok = r.check("test", params)
	if ok {
		t.Fatal("Match ok when it should not be")
	}
}
