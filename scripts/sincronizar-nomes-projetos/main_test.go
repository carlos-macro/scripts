package main

import "testing"

func TestSepararIDEMontarNome(t *testing.T) {
	casos := []struct {
		projectID string
		ok        bool
		nome      string
	}{
		{"4918426000052796600", true, "Oi"},
		{"4918426000052796600-DEV", true, "Oi (DEV)"},
		{"4918426000052796600-IA-DEV", true, "Oi (IA-DEV)"},
		{"PRJ-REFIDEL", false, ""},
		{"Macro teste", false, ""},
		{"-VOZ", false, ""},
	}
	for _, c := range casos {
		_, sufixo, ok := separarID(c.projectID)
		if ok != c.ok {
			t.Errorf("separarID(%q) ok = %v, esperado %v", c.projectID, ok, c.ok)
			continue
		}
		if ok {
			if nome := montarNome("Oi", sufixo); nome != c.nome {
				t.Errorf("montarNome para %q = %q, esperado %q", c.projectID, nome, c.nome)
			}
		}
	}
}
