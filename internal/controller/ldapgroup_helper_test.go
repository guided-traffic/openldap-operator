/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

func TestLDAPGroupControllerHelper_MemberDNResolution(t *testing.T) {
	reconciler := &LDAPGroupReconciler{}

	tests := []struct {
		name        string
		input       []string
		baseDN      string
		expectedOut []string
	}{
		{
			name:   "simple usernames",
			input:  []string{"alice", "bob"},
			baseDN: "dc=example,dc=com",
			expectedOut: []string{
				"uid=alice,ou=users,dc=example,dc=com",
				"uid=bob,ou=users,dc=example,dc=com",
			},
		},
		{
			name:   "mixed DNs and usernames",
			input:  []string{"alice", "uid=bob,ou=people,dc=example,dc=com"},
			baseDN: "dc=example,dc=com",
			expectedOut: []string{
				"uid=alice,ou=users,dc=example,dc=com",
				"uid=bob,ou=people,dc=example,dc=com",
			},
		},
		{
			name:   "all DNs",
			input:  []string{"uid=alice,ou=people,dc=example,dc=com", "cn=admin,dc=example,dc=com"},
			baseDN: "dc=example,dc=com",
			expectedOut: []string{
				"uid=alice,ou=people,dc=example,dc=com",
				"cn=admin,dc=example,dc=com",
			},
		},
		{
			name:        "empty input",
			input:       []string{},
			baseDN:      "dc=example,dc=com",
			expectedOut: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ldapServer := &openldapv1.LDAPServer{
				Spec: openldapv1.LDAPServerSpec{
					BaseDN: tt.baseDN,
				},
			}

			result := reconciler.resolveMemberDNs(ldapServer, tt.input)
			assert.Equal(t, tt.expectedOut, result)
		})
	}
}

func TestLDAPGroupControllerHelper_MemberDNResolutionWithNilServer(t *testing.T) {
	reconciler := &LDAPGroupReconciler{}

	input := []string{"alice", "bob"}
	result := reconciler.resolveMemberDNs(nil, input)

	// Should return input unchanged when server is nil
	assert.Equal(t, input, result)
}
