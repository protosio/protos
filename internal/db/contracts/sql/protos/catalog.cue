package protos

catalog: {
	package:    "protoscontracts"
	var_name:   "Catalog"
	surface_id: "protos.db"
	initial_alias: "v00"
	initial_create_sql_var: "CreateSQL"
	versions: [{
		alias:       "v00"
		import_path: "github.com/protosio/protos/internal/db/contracts/sql/protos/v0_0"
		version_var: "Version"
		transition_var: "Transition"
		create_sql_var: "CreateSQL"
	}]
}
