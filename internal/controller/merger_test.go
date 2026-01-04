package controller

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merger", func() {
	DescribeTable("Merge",
		func(a string, b string, expected string) {
			as := make(map[string]interface{})
			_ = json.Unmarshal([]byte(a), &as)

			bs := make(map[string]interface{})
			_ = json.Unmarshal([]byte(b), &bs)
			merge := MergeMaps(as, bs)

			result, _ := json.Marshal(merge)
			Expect(string(result)).To(Equal(expected))
		},
		Entry("Empty Maps", "{}", "{}", "{}"),
		Entry("Default map a", "{\"test\":\"value\"}", "{}", "{\"test\":\"value\"}"),
		Entry("Default map b", "{}", "{\"test\":\"value\"}", "{\"test\":\"value\"}"),
		Entry("Override map a", "{\"test\":\"original\"}", "{\"test\":\"replaced\"}", "{\"test\":\"replaced\"}"),
		Entry("Default array a", "{\"test\":[{\"test\":\"value\"}]}", "{}", "{\"test\":[{\"test\":\"value\"}]}"),
		Entry("Default array b", "{}", "{\"test\":[{\"test\":\"value\"}]}", "{\"test\":[{\"test\":\"value\"}]}"),
		Entry("Override array a", "{\"test\":[{\"test\":\"original\"}]}", "{\"test\":[{\"test\":\"replaced\"}]}", "{\"test\":[{\"test\":\"replaced\"}]}"),
	)
})
