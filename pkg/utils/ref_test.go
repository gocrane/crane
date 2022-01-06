package utils

import "testing"

type KeyTest struct {
	input  string
	output string
}

func TestGetContainerIdFromKey(t *testing.T) {
	var cases = []KeyTest{
		{
			input:  "/kubepods/besteffort/pod04e5e9e7-8d95-44dd-9af7-ab944405fff8/18b514fc91ecb19b7ee79ebeaa6f2df86c6c939e420520b97ad4f7532582d35a",
			output: "18b514fc91ecb19b7ee79ebeaa6f2df86c6c939e420520b97ad4f7532582d35a",
		},
		{
			input:  "/kubepods/besteffort/pod04e5e9e7-8d95-44dd-9af7-ab944405fff8",
			output: "",
		},
		{
			input:  "/kubepods/besteffort/pod04e5e9e7-8d95-44dd-9af7-ab944405fff8/2cc2c4badac0618edda11bdd06826e7385b885ca88323b6f5d90270395e039d9",
			output: "2cc2c4badac0618edda11bdd06826e7385b885ca88323b6f5d90270395e039d9",
		},
	}

	for _, c := range cases {
		if r := GetContainerIdFromKey(c.input); r != c.output {
			t.Fatalf("TestGetContainerIdFromKey failed {%s,%s}, r: %s", c.input, c.output, r)
		}
	}

	return
}
