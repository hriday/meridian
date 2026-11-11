package tfscan

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

const planFixture = `{
  "format_version": "1.2",
  "terraform_version": "1.7.0",
  "planned_values": {
    "root_module": {
      "resources": [
        {
          "address": "aws_security_group.web",
          "mode": "managed",
          "type": "aws_security_group",
          "name": "web",
          "provider_name": "registry.terraform.io/hashicorp/aws",
          "values": {
            "name": "web-sg",
            "ingress": [
              { "from_port": 22, "to_port": 22, "protocol": "tcp", "cidr_blocks": ["0.0.0.0/0"] }
            ]
          }
        }
      ],
      "child_modules": [
        {
          "address": "module.database",
          "resources": [
            {
              "address": "module.database.aws_db_instance.main",
              "type": "aws_db_instance",
              "name": "main",
              "provider_name": "registry.terraform.io/hashicorp/aws",
              "values": { "publicly_accessible": true, "password": "PLANSECRETpw", "storage_encrypted": false }
            }
          ]
        }
      ]
    }
  }
}`

func TestPlanParsesNestedModules(t *testing.T) {
	rs, err := parsePlanJSON(planFixture)
	require.NoError(t, err)
	require.Len(t, rs, 2) // root SG + child-module db
	types := map[string]bool{}
	for _, r := range rs {
		types[r.typ] = true
		require.Equal(t, "aws", r.provider)
	}
	require.True(t, types["aws_security_group"])
	require.True(t, types["aws_db_instance"])
}

func TestPlanEndToEndFindings(t *testing.T) {
	rs, err := parsePlanJSON(planFixture)
	require.NoError(t, err)
	a := check(rs)
	require.NotEmpty(t, findingsWith(a, "22"))
	require.NotEmpty(t, findingsWith(a, "publicly accessible"))
	require.NotEmpty(t, findingsWith(a, "encrypt"))
}

func TestPlanSecretNotSurfaced(t *testing.T) {
	rs, err := parsePlanJSON(planFixture)
	require.NoError(t, err)
	a := check(rs)
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "PLANSECRETpw")
}

func TestPlanNonPlanJSON(t *testing.T) {
	_, err := parsePlanJSON(`{"foo":1}`)
	require.Error(t, err)
}
