package httpapi

import (
	"strings"
	"testing"

	"github.com/cznUSER/B-HIS/internal/store/seed"
)

func TestAccessCatalogHTTPCreateAndPatch(t *testing.T) {
	ts, _, issuer := newMasterHTTP(t)
	tok := bearerFor(t, issuer, seed.DemoUser())
	api := ts.URL + "/api/v1/access"

	code, body, _ := postJSON(t, api+"/modules", `{"code":"clinical-meta","name":"临床配置","departmentFunctions":"9007199254740993","needSpecifyDepartment":true}`, authHdr(tok))
	if code != 201 || !strings.Contains(string(body), `"departmentFunctions":"9007199254740993"`) {
		t.Fatalf("create module %d %s", code, body)
	}
	code, body = patchJSON(t, api+"/modules/clinical-meta", `{"homePageUrl":"/clinical/home"}`, authHdr(tok))
	if code != 200 || !strings.Contains(string(body), `"homePageUrl":"/clinical/home"`) {
		t.Fatalf("patch module %d %s", code, body)
	}

	code, body, _ = postJSON(t, api+"/categories", `{"id":"clinical","name":"临床"}`, authHdr(tok))
	if code != 201 || !strings.Contains(string(body), `"id":"clinical"`) {
		t.Fatalf("create category %d %s", code, body)
	}
	code, body = patchJSON(t, api+"/categories/clinical", `{"status":"INACTIVE"}`, authHdr(tok))
	if code != 200 || !strings.Contains(string(body), `"status":"INACTIVE"`) {
		t.Fatalf("patch category %d %s", code, body)
	}

	code, body, _ = postJSON(t, api+"/modes", `{"code":"card-pin","name":"卡+PIN"}`, authHdr(tok))
	if code != 201 || !strings.Contains(string(body), `"status":"ACTIVE"`) {
		t.Fatalf("create mode %d %s", code, body)
	}
	code, body = patchJSON(t, api+"/modes/card-pin", `{"status":"DISABLED"}`, authHdr(tok))
	if code != 200 || !strings.Contains(string(body), `"status":"DISABLED"`) {
		t.Fatalf("patch mode %d %s", code, body)
	}
}
