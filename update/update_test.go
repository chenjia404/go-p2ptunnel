package update

import "testing"

func TestVerifySignature(t *testing.T) {
	if v, e := VerifySignature("1.zip"); v != true {
		t.Errorf(e.Error())
		t.Errorf("VerifySignature")
	}
}
