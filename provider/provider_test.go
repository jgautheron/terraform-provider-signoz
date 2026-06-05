// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories instantiates the provider for acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"signoz": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies the env needed to talk to a live SigNoz is present.
// Acceptance tests only run when TF_ACC is set.
func testAccPreCheck(t *testing.T) {
	if os.Getenv("SIGNOZ_ENDPOINT") == "" {
		t.Fatal("SIGNOZ_ENDPOINT must be set for acceptance tests")
	}
	if os.Getenv("SIGNOZ_ACCESS_TOKEN") == "" {
		t.Fatal("SIGNOZ_ACCESS_TOKEN must be set for acceptance tests")
	}
}
