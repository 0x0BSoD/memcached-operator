package v1

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Memcached Webhook", func() {

	Context("When creating Memcached under Defaulting Webhook", func() {
		It("Should fill in the default value if a required field is empty", func() {

			// TODO(user): Add your logic here

		})
	})

	Context("When creating Memcached under Validating Webhook", func() {
		It("Should deny if a required field is empty", func() {

			// TODO(user): Add your logic here

		})

		It("Should admit if all required fields are provided", func() {

			// TODO(user): Add your logic here

		})
	})

})
