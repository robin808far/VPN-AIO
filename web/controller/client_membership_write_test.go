package controller

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/xray"
)

func blockMembershipInsert(t *testing.T, inboundId int) {
	t.Helper()
	statement := fmt.Sprintf(`CREATE TRIGGER reject_membership_%d BEFORE INSERT ON account_inbounds
		WHEN NEW.inbound_id = %d BEGIN SELECT RAISE(ABORT, 'membership insert blocked by test'); END`, inboundId, inboundId)
	if err := database.GetDB().Exec(statement).Error; err != nil {
		t.Fatalf("create membership failure trigger: %v", err)
	}
}

func assertClientMembershipPartialFailure(t *testing.T, msg map[string]any) {
	t.Helper()
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("membership failure was reported as success: %v", msg)
	}
	text, _ := msg["msg"].(string)
	if !strings.Contains(text, "client data was saved") || !strings.Contains(text, "membership updates failed") {
		t.Fatalf("response did not explain the partial write: %q", text)
	}
}

func grantTestResellerInbound(t *testing.T, resellerId, inboundId int) {
	t.Helper()
	if err := database.GetDB().Create(&model.InboundAccess{UserId: resellerId, InboundId: inboundId}).Error; err != nil {
		t.Fatalf("grant reseller inbound %d: %v", inboundId, err)
	}
}

func TestAddClientPreflightsAllRequestedMemberships(t *testing.T) {
	env := newBulkMembershipEnv(t)
	primary := seedMembershipInbound(t, env.admin.Id, model.VLESS, 48901, "")
	psk := &model.Inbound{
		UserId: env.admin.Id, Tag: "ikev2-48902", Port: 48902, Protocol: model.IKEV2, Enable: true,
		Settings: `{"authMode":"psk","clients":[{"email":"already@example.com","password":"pw","enable":true}]}`,
	}
	if err := database.GetDB().Create(psk).Error; err != nil {
		t.Fatalf("create PSK inbound: %v", err)
	}
	syncMembershipAccounts(t)

	const email = "preflight@example.com"
	msg := bulkMembershipPost(t, env.admin, "/panel/api/inbounds/addClient", clientBody(t, primary.Id, map[string]any{
		"id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "email": email, "enable": false,
	}, []int{primary.Id, psk.Id}, false))
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("the occupied PSK membership was accepted: %v", msg)
	}
	if inboundHolds(t, primary.Id, email) {
		t.Error("client settings were written before the membership admission refusal")
	}
	var trafficRows int64
	if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("email = ?", email).Count(&trafficRows).Error; err != nil {
		t.Fatalf("count client traffic rows: %v", err)
	}
	if trafficRows != 0 {
		t.Errorf("created %d traffic rows before membership admission succeeded", trafficRows)
	}
	account, err := accountService.GetAccountByEmail(email)
	if err != nil || account != nil {
		t.Errorf("account state after preflight refusal = %+v (err %v), want no account", account, err)
	}
}

func TestAddClientBatchPreflightsPskCapacityAsASet(t *testing.T) {
	env := newBulkMembershipEnv(t)
	psk := &model.Inbound{
		UserId: env.admin.Id, Tag: "ikev2-48903", Port: 48903, Protocol: model.IKEV2, Enable: true,
		Settings: `{"authMode":"psk","clients":[]}`,
	}
	if err := database.GetDB().Create(psk).Error; err != nil {
		t.Fatalf("create PSK inbound: %v", err)
	}

	form := url.Values{}
	form.Set("id", strconv.Itoa(psk.Id))
	form.Set("settings", `{"clients":[`+
		`{"id":"one","email":"batch-one@example.com","enable":false},`+
		`{"id":"two","email":"batch-two@example.com","enable":false}]}`)
	msg := bulkMembershipPost(t, env.admin, "/panel/api/inbounds/addClient", form.Encode())
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("a two-account batch was admitted to a single-account PSK inbound: %v", msg)
	}
	for _, email := range []string{"batch-one@example.com", "batch-two@example.com"} {
		if inboundHolds(t, psk.Id, email) {
			t.Errorf("batch account %s was saved before PSK capacity refusal", email)
		}
	}
}

func TestAddClientReportsPostWriteMembershipFailureAndKeepsResellerCharge(t *testing.T) {
	f := newResellerFixture(t)
	extra := seedMembershipInbound(t, f.ali.Id, model.Trojan, 48911, "")
	grantTestResellerInbound(t, f.sara.Id, extra.Id)
	syncMembershipAccounts(t)
	blockMembershipInsert(t, extra.Id)
	spentBefore := f.spent(t)

	const email = "new-sold-client@example.com"
	msg := bulkMembershipPost(t, f.sara, "/panel/api/inbounds/addClient", clientBody(t, f.aliInbound.Id, map[string]any{
		"id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "email": email, "enable": false,
		"totalGB": float64(testGB),
	}, []int{f.aliInbound.Id, extra.Id}, false))
	assertClientMembershipPartialFailure(t, msg)

	if !inboundHolds(t, f.aliInbound.Id, email) {
		t.Fatal("the primary inbound write was lost after membership projection failed")
	}
	if inboundHolds(t, extra.Id, email) {
		t.Error("the refused secondary membership was written")
	}
	ids, err := accountService.InboundIdsForEmail(email)
	if err != nil || len(ids) != 1 || ids[0] != f.aliInbound.Id {
		t.Errorf("recovered memberships = %v (err %v), want only the saved primary inbound", ids, err)
	}
	owner := &model.ResellerClient{}
	if err := database.GetDB().Where("email = ?", email).First(owner).Error; err != nil {
		t.Fatalf("reseller ownership charge was rolled back: %v", err)
	}
	if owner.UserId != f.sara.Id || owner.ChargedBytes != testGB {
		t.Errorf("reseller charge = %+v, want seller %d charged %d bytes", owner, f.sara.Id, testGB)
	}
	if got := f.spent(t); got != spentBefore+testGB {
		t.Errorf("reseller spent = %d, want %d; the saved account's reservation must remain", got, spentBefore+testGB)
	}
}

func TestUpdateClientReportsPostWriteMembershipFailureAndKeepsQuotaCharge(t *testing.T) {
	f := newResellerFixture(t)
	extra := seedMembershipInbound(t, f.ali.Id, model.Trojan, 48921, "")
	grantTestResellerInbound(t, f.sara.Id, extra.Id)
	syncMembershipAccounts(t)
	blockMembershipInsert(t, extra.Id)
	spentBefore := f.spent(t)

	msg := bulkMembershipPost(t, f.sara, "/panel/api/inbounds/updateClient/sara-uuid", clientBody(t, f.aliInbound.Id, map[string]any{
		"id": "sara-uuid", "email": f.saraEmail, "enable": false,
		"totalGB": float64(11 * testGB), "comment": "quota updated before membership refusal",
	}, []int{f.aliInbound.Id, extra.Id}, false))
	assertClientMembershipPartialFailure(t, msg)

	if !inboundHolds(t, f.aliInbound.Id, f.saraEmail) {
		t.Fatal("the primary inbound edit was lost after membership projection failed")
	}
	if inboundHolds(t, extra.Id, f.saraEmail) {
		t.Error("the refused secondary membership was written")
	}
	owner := &model.ResellerClient{}
	if err := database.GetDB().Where("email = ?", f.saraEmail).First(owner).Error; err != nil {
		t.Fatalf("reseller ownership row was lost: %v", err)
	}
	if owner.ChargedBytes != 11*testGB {
		t.Errorf("charged quota = %d, want %d; a saved quota increase must remain charged", owner.ChargedBytes, 11*testGB)
	}
	if got := f.spent(t); got != spentBefore+testGB {
		t.Errorf("reseller spent = %d, want %d", got, spentBefore+testGB)
	}
	traffic := accountTraffic(t, f.saraEmail)
	if traffic.Total != 11*testGB {
		t.Errorf("saved quota = %d, want %d", traffic.Total, 11*testGB)
	}
}

func TestUpdateClientPreflightsNewMembershipBeforeResellerCharge(t *testing.T) {
	f := newResellerFixture(t)
	psk := &model.Inbound{
		UserId: f.ali.Id, Tag: "ikev2-48931", Port: 48931, Protocol: model.IKEV2, Enable: true,
		Settings: `{"authMode":"psk","clients":[{"email":"other@example.com","password":"pw","enable":true}]}`,
	}
	if err := database.GetDB().Create(psk).Error; err != nil {
		t.Fatalf("create PSK inbound: %v", err)
	}
	grantTestResellerInbound(t, f.sara.Id, psk.Id)
	syncMembershipAccounts(t)
	spentBefore := f.spent(t)

	msg := bulkMembershipPost(t, f.sara, "/panel/api/inbounds/updateClient/sara-uuid", clientBody(t, f.aliInbound.Id, map[string]any{
		"id": "sara-uuid", "email": f.saraEmail, "enable": false,
		"totalGB": float64(11 * testGB), "comment": "must not persist",
	}, []int{f.aliInbound.Id, psk.Id}, false))
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("the occupied PSK membership was accepted: %v", msg)
	}
	if inboundHolds(t, psk.Id, f.saraEmail) {
		t.Error("the refused membership was written")
	}
	var stored model.Inbound
	if err := database.GetDB().Where("id = ?", f.aliInbound.Id).First(&stored).Error; err != nil {
		t.Fatalf("reload primary inbound: %v", err)
	}
	if strings.Contains(stored.Settings, "must not persist") {
		t.Error("the primary account edit ran before new-membership admission")
	}
	owner := &model.ResellerClient{}
	if err := database.GetDB().Where("email = ?", f.saraEmail).First(owner).Error; err != nil {
		t.Fatalf("reload reseller ownership: %v", err)
	}
	if owner.ChargedBytes != 10*testGB || f.spent(t) != spentBefore {
		t.Errorf("preflight refusal changed reseller charge: charged=%d spent=%d, want charged=%d spent=%d",
			owner.ChargedBytes, f.spent(t), 10*testGB, spentBefore)
	}
}

func TestUpdateClientReportsAccountRenameFailureWithoutCreatingDuplicate(t *testing.T) {
	env := newBulkMembershipEnv(t)
	const oldEmail = "rename-old@example.com"
	const newEmail = "rename-new@example.com"
	const clientID = "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	primary := seedMembershipInbound(t, env.admin.Id, model.VLESS, 48941,
		`{"id":"`+clientID+`","email":"`+oldEmail+`","enable":true}`)
	secondary := seedMembershipInbound(t, env.admin.Id, model.Trojan, 48942,
		`{"password":"rename-password","email":"`+oldEmail+`","enable":true}`)
	syncMembershipAccounts(t)
	if err := database.GetDB().Exec(`CREATE TRIGGER reject_account_rename BEFORE UPDATE OF email ON accounts
		WHEN OLD.email = 'rename-old@example.com' AND NEW.email = 'rename-new@example.com'
		BEGIN SELECT RAISE(ABORT, 'account rename blocked by test'); END`).Error; err != nil {
		t.Fatalf("create account rename trigger: %v", err)
	}

	msg := bulkMembershipPost(t, env.admin, "/panel/api/inbounds/updateClient/"+clientID, clientBody(t, primary.Id, map[string]any{
		"id": clientID, "email": newEmail, "enable": true, "comment": "renamed account",
	}, []int{primary.Id, secondary.Id}, false))
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("failed account rename was reported as success: %v", msg)
	}
	text, _ := msg["msg"].(string)
	if !strings.Contains(text, "client data was saved") || !strings.Contains(text, "account rename failed") {
		t.Fatalf("response did not report the saved edit and failed account rename: %q", text)
	}
	if !inboundHolds(t, primary.Id, newEmail) || !inboundHolds(t, secondary.Id, oldEmail) || inboundHolds(t, secondary.Id, newEmail) {
		t.Error("rename failure continued into membership projection and split the account")
	}
	oldAccount, err := accountService.GetAccountByEmail(oldEmail)
	if err != nil || oldAccount == nil {
		t.Errorf("old account after failed rename = %+v (err %v), want it preserved", oldAccount, err)
	}
	newAccount, err := accountService.GetAccountByEmail(newEmail)
	if err != nil || newAccount != nil {
		t.Errorf("unexpected duplicate account after failed rename = %+v (err %v)", newAccount, err)
	}
}

func TestUpdateClientReportsResellerRenameFailureAndKeepsCharge(t *testing.T) {
	f := newResellerFixture(t)
	syncMembershipAccounts(t)
	const newEmail = "sara-renamed@example.com"
	if err := database.GetDB().Exec(`CREATE TRIGGER reject_reseller_rename BEFORE UPDATE OF email ON reseller_clients
		WHEN OLD.email = 'sara-client' AND NEW.email = 'sara-renamed@example.com'
		BEGIN SELECT RAISE(ABORT, 'reseller rename blocked by test'); END`).Error; err != nil {
		t.Fatalf("create reseller rename trigger: %v", err)
	}
	spentBefore := f.spent(t)

	msg := bulkMembershipPost(t, f.sara, "/panel/api/inbounds/updateClient/sara-uuid", clientBody(t, f.aliInbound.Id, map[string]any{
		"id": "sara-uuid", "email": newEmail, "enable": false, "totalGB": float64(10 * testGB),
	}, []int{f.aliInbound.Id}, false))
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("failed reseller ownership rename was reported as success: %v", msg)
	}
	text, _ := msg["msg"].(string)
	if !strings.Contains(text, "client data was saved") || !strings.Contains(text, "reseller ownership rename failed") {
		t.Fatalf("response did not report the saved edit and failed ledger rename: %q", text)
	}
	if !inboundHolds(t, f.aliInbound.Id, newEmail) {
		t.Error("the primary client edit was not retained")
	}
	oldOwner, err := (&service.ResellerService{}).ClientOwner(f.saraEmail)
	if err != nil || oldOwner == nil || oldOwner.ChargedBytes != 10*testGB {
		t.Errorf("old reseller charge after failed rename = %+v (err %v), want the existing charge retained", oldOwner, err)
	}
	newOwner, err := (&service.ResellerService{}).ClientOwner(newEmail)
	if err != nil || newOwner != nil {
		t.Errorf("failed rename created a new ledger owner: %+v (err %v)", newOwner, err)
	}
	if got := f.spent(t); got != spentBefore {
		t.Errorf("reseller spent changed on rename failure: got %d, want %d", got, spentBefore)
	}
}

func TestUpdateClientRestoresResellerOwnershipIfAccountRenameFails(t *testing.T) {
	f := newResellerFixture(t)
	syncMembershipAccounts(t)
	const newEmail = "sara-account-rename-failed@example.com"
	if err := database.GetDB().Exec(`CREATE TRIGGER reject_reseller_account_rename BEFORE UPDATE OF email ON accounts
		WHEN OLD.email = 'sara-client' AND NEW.email = 'sara-account-rename-failed@example.com'
		BEGIN SELECT RAISE(ABORT, 'account rename blocked by test'); END`).Error; err != nil {
		t.Fatalf("create account rename trigger: %v", err)
	}
	spentBefore := f.spent(t)

	msg := bulkMembershipPost(t, f.sara, "/panel/api/inbounds/updateClient/sara-uuid", clientBody(t, f.aliInbound.Id, map[string]any{
		"id": "sara-uuid", "email": newEmail, "enable": false, "totalGB": float64(10 * testGB),
	}, []int{f.aliInbound.Id}, false))
	if success, _ := msg["success"].(bool); success {
		t.Fatalf("failed account rename was reported as success: %v", msg)
	}
	text, _ := msg["msg"].(string)
	if !strings.Contains(text, "account rename failed") {
		t.Fatalf("response did not report the account rename failure: %q", text)
	}
	oldOwner, err := (&service.ResellerService{}).ClientOwner(f.saraEmail)
	if err != nil || oldOwner == nil || oldOwner.ChargedBytes != 10*testGB {
		t.Errorf("old reseller ownership was not restored: %+v (err %v)", oldOwner, err)
	}
	newOwner, err := (&service.ResellerService{}).ClientOwner(newEmail)
	if err != nil || newOwner != nil {
		t.Errorf("new reseller ownership remains after failed account rename: %+v (err %v)", newOwner, err)
	}
	if got := f.spent(t); got != spentBefore {
		t.Errorf("spent bytes changed on failed rename: got %d, want %d", got, spentBefore)
	}
	oldAccount, err := accountService.GetAccountByEmail(f.saraEmail)
	if err != nil || oldAccount == nil {
		t.Errorf("old account after failed rename = %+v (err %v), want it preserved", oldAccount, err)
	}
	newAccount, err := accountService.GetAccountByEmail(newEmail)
	if err != nil || newAccount != nil {
		t.Errorf("failed rename created another account: %+v (err %v)", newAccount, err)
	}
}
