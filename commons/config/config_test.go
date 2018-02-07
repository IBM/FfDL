package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDataStoreConfig(t *testing.T) {
	c := GetDataStoreConfig()
	assert.Contains(t, c, "auth_url")
	assert.Contains(t, c, "user_name")
	assert.Contains(t, c, "password")
	assert.Contains(t, c, "type")

}
