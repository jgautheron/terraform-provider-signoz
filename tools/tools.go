// Copyright (c) Jonathan Gautheron
// SPDX-License-Identifier: Apache-2.0

//go:build generate

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

// Format Terraform examples used in documentation.
//go:generate terraform fmt -recursive ../examples/

// Generate provider documentation from schema descriptions + examples/.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name signoz
