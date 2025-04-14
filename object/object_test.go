package object

import (
	"testing"
)

func TestStringDictKey(t *testing.T) {
	hello1 := &String{Value: "Hello World"}
	hello2 := &String{Value: "Hello World"}
	diff1 := &String{Value: "My name is johnny"}
	diff2 := &String{Value: "My name is johnny"}

	if hello1.DictKey() != hello2.DictKey() {
		t.Errorf("hello1.DictKey() != hello2.DictKey()")
	}

	if diff1.DictKey() != diff2.DictKey() {
		t.Errorf("diff1.DictKey() != diff2.DictKey()")
	}
}
