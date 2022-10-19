package conjurapi

import (
	"os"
	"strings"
	"testing"

	"github.com/Nirupma-Verma/conjur-api-go/conjurapi/authn"
	"github.com/stretchr/testify/assert"
)

func v5Setup() (*Client, error) {
	config := &Config{}
	config.mergeEnv()

	apiKey := os.Getenv("CONJUR_AUTHN_API_KEY")
	login := os.Getenv("CONJUR_AUTHN_LOGIN")

	policy := `
- !user alice
- !host bob

- !variable db-password
- !variable db-password-2
- !variable password

- !permit
  role: !user alice
  privilege: [ execute ]
  resource: !variable db-password

- !policy
  id: prod
  body:
  - !variable cluster-admin
  - !variable cluster-admin-password

  - !policy
    id: database
    body:
    - !variable username
    - !variable password
`

	conjur, err := NewClientFromKey(*config, authn.LoginPair{Login: login, APIKey: apiKey})

	if err == nil {
		conjur.LoadPolicy(
			PolicyModePut,
			"root",
			strings.NewReader(policy),
		)
	}

	return conjur, err
}

func v4Setup() (*Client, error) {
	config := &Config{
		ApplianceURL: os.Getenv("CONJUR_V4_APPLIANCE_URL"),
		SSLCert:      os.Getenv("CONJUR_V4_SSL_CERTIFICATE"),
		Account:      os.Getenv("CONJUR_V4_ACCOUNT"),
		V4:           true,
	}

	login := os.Getenv("CONJUR_V4_AUTHN_LOGIN")
	apiKey := os.Getenv("CONJUR_V4_AUTHN_API_KEY")

	return NewClientFromKey(*config, authn.LoginPair{Login: login, APIKey: apiKey})
}

func TestClient_CheckPermission(t *testing.T) {
	checkAllowed := func(conjur *Client, id string) func(t *testing.T) {
		return func(t *testing.T) {
			allowed, err := conjur.CheckPermission(id, "execute")

			assert.NoError(t, err)
			assert.True(t, allowed)
		}
	}

	checkNonExisting := func(conjur *Client, id string) func(t *testing.T) {
		return func(t *testing.T) {
			allowed, err := conjur.CheckPermission(id, "execute")

			assert.NoError(t, err)
			assert.False(t, allowed)
		}
	}

	t.Run("V5", func(t *testing.T) {
		conjur, err := v5Setup()
		assert.NoError(t, err)

		t.Run("Check an allowed permission", checkAllowed(conjur, "cucumber:variable:db-password"))

		t.Run("Check a permission on a non-existent resource", checkNonExisting(conjur, "cucumber:variable:foobar"))
	})

	if os.Getenv("TEST_VERSION") != "oss" {
		t.Run("V4", func(t *testing.T) {
			conjur, err := v4Setup()
			assert.NoError(t, err)

			t.Run("Check an allowed permission", checkAllowed(conjur, "cucumber:variable:existent-variable-with-defined-value"))

			t.Run("Check a permission on a non-existent resource", checkNonExisting(conjur, "cucumber:variable:foobar"))
		})
	}
}

func TestClient_Resources(t *testing.T) {
	listResources := func(conjur *Client, filter *ResourceFilter, expected int) func(t *testing.T) {
		return func(t *testing.T) {
			resources, err := conjur.Resources(filter)
			assert.NoError(t, err)
			assert.Len(t, resources, expected)
		}
	}

	t.Run("V5", func(t *testing.T) {
		conjur, err := v5Setup()
		assert.NoError(t, err)

		t.Run("Lists all resources", listResources(conjur, nil, 12))
		t.Run("Lists resources by kind", listResources(conjur, &ResourceFilter{Kind: "variable"}, 7))
		t.Run("Lists resources that start with db", listResources(conjur, &ResourceFilter{Search: "db"}, 2))
		t.Run("Lists variables that start with prod/database", listResources(conjur, &ResourceFilter{Search: "prod/database", Kind: "variable"}, 2))
		t.Run("Lists variables that start with prod", listResources(conjur, &ResourceFilter{Search: "prod", Kind: "variable"}, 4))
		t.Run("Lists resources and limit result to 1", listResources(conjur, &ResourceFilter{Limit: 1}, 1))
		t.Run("Lists resources after the first", listResources(conjur, &ResourceFilter{Offset: 1}, 10))
	})

	if os.Getenv("TEST_VERSION") != "oss" {
		t.Run("V4", func(t *testing.T) {
			_, err := v4Setup()
			assert.NoError(t, err)
			// v4 router doesn't support it showResource
		})
	}
}

func TestClient_Resource(t *testing.T) {
	showResource := func(conjur *Client, id string) func(t *testing.T) {
		return func(t *testing.T) {
			_, err := conjur.Resource(id)
			assert.NoError(t, err)
		}
	}

	t.Run("V5", func(t *testing.T) {
		conjur, err := v5Setup()
		assert.NoError(t, err)

		t.Run("Shows a resource", showResource(conjur, "cucumber:variable:db-password"))
	})

	if os.Getenv("TEST_VERSION") != "oss" {
		t.Run("V4", func(t *testing.T) {
			_, err := v4Setup()
			assert.NoError(t, err)
			// v4 router doesn't support it showResource
		})
	}
}
