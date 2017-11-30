package auth

import (
	ldap "github.com/vjeantet/ldapserver"
)

// LDAPsrv starts an LDAP server
func LDAPsrv() {
	log.Info("Starting LDAP server")
	//ldap logger
	ldap.Logger = log
	//Create a new LDAP Server
	server := ldap.NewServer()
	routes := ldap.NewRouteMux()
	routes.Bind(handleBind)
	server.Handle(routes)
	// listen on 10389
	server.ListenAndServe("0.0.0.0:10389")
}

func handleBind(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetBindRequest()
	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)

	_, err := ValidateAndGetUser(string(r.Name()), string(r.AuthenticationSimple()))
	if err == nil {
		w.Write(res)
		return
	}

	log.Printf("Bind failed User=%s, Pass=%s", string(r.Name()), string(r.AuthenticationSimple()))
	res.SetResultCode(ldap.LDAPResultInvalidCredentials)
	res.SetDiagnosticMessage("Invalid credentials")
	w.Write(res)
}
