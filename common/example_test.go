package common_test

import (
	"encoding/json"
	"fmt"

	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

func ExamplePtr() {
	// Remark is sent as an empty string; other optional fields are omitted.
	data, err := json.Marshal(cloudphone.UpdateItem{
		ImageID: "phone-1",
		Remark:  common.Ptr(""),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	// Output: {"image_id":"phone-1","remark":""}
}
