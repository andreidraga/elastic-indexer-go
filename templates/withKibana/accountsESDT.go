package withKibana

var AccountsESDT = Object{
	"index_patterns": Array{
		"accountsesdt-*",
	},
	"settings": Object{
		"number_of_shards":                                 3,
		"number_of_replicas":                               0,
		"opendistro.index_state_management.policy_id":      "accountsesdt_policy",
		"opendistro.index_state_management.rollover_alias": "accountsesdt",
	},
	"mappings": Object{
		"properties": Object{
			"balanceNum": Object{
				"type": "double",
			},
		},
	},
}
