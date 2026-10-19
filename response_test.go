package apex

import "testing"

func TestResponsePayload(t *testing.T) {
	var payload = &struct{ Name string }{Name: "Ada"}
	var response = NewResponse(201, "application/custom", payload)
	if response.StatusCode != 201 || response.ContentType != "application/custom" || response.Value != payload || response.Adopted {
		t.Fatalf("response = %#v", response)
	}

	var adopted = NewAdopedtResponse()
	if !adopted.Adopted || adopted.Value != nil || adopted.StatusCode != 0 || adopted.ContentType != "" {
		t.Fatalf("adopted response = %#v", adopted)
	}
}
