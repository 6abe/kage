package input

import (
	"reflect"
	"testing"
)

func TestAllowed(t *testing.T) {
	if Allowed(false, "", false) {
		t.Fatal("denied")
	}
	if !Allowed(true, "", false) || !Allowed(false, "1", false) || !Allowed(false, "", true) {
		t.Fatal("allowed")
	}
	if Allowed(false, "true", false) {
		t.Fatal("only KAGE_ALLOW_INPUT=1 counts")
	}
}

func TestTypeArgsClear(t *testing.T) {
	got := TypeArgs("hi", true)
	want := []string{"-M", "ctrl", "-k", "a", "-m", "ctrl", "--", "hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	got = TypeArgs("", true)
	want = []string{"-M", "ctrl", "-k", "a", "-m", "ctrl", "-k", "BackSpace"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
}

func TestCanonicalKey(t *testing.T) {
	k, err := CanonicalKey("return")
	if err != nil || k != "Return" {
		t.Fatalf("%s %v", k, err)
	}
	if _, err := CanonicalKey("F1"); err == nil {
		t.Fatal("F1")
	}
}
