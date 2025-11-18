package serviceaccount

import (
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/json-iterator/go"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/wisecloudtool/types"
)

func TestCipherSuitesFromName(t *testing.T) {
	convey.Convey("Test cipherSuitesFromName", t, func() {
		convey.Convey("success", func() {
			cipherSuitesArr := []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"}
			tlsSuite := cipherSuitesID(cipherSuitesFromName(cipherSuitesArr))
			convey.So(len(tlsSuite), convey.ShouldEqual, 2)
		})
	})
}
