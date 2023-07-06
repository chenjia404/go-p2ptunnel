package update

import "testing"

func TestVerifySignature(t *testing.T) {
	if v, e := VerifySignature("go-p2ptunnel_0.1.19_darwin_amd64.tar.gz"); v != true {
		t.Errorf(e.Error())
		t.Errorf("VerifySignature")
	}
}
