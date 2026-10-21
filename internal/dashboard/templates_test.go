package dashboard

import "testing"

func TestTemplatesAreValidKeyFreeAndRegenerateIdentity(t *testing.T) {
	for _, definition := range templateDefinitions {
		first, err := templateBoard(definition.ID)
		if err != nil {
			t.Fatal(err)
		}
		second, err := templateBoard(definition.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err = first.Validate(); err != nil {
			t.Fatal(err)
		}
		if first.Widgets[0].ID == second.Widgets[0].ID {
			t.Fatal("template reuses widget identity")
		}
		if !definition.RequiresNetwork && len(first.Sources) != 0 {
			t.Fatal("offline template includes network source")
		}
	}
}
