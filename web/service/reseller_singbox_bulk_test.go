package service

import (
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/xray"
	"gorm.io/gorm"
)

func TestApplyBulkChargesUpdatesSingboxAccountAndTrafficPolicy(t *testing.T) {
	singboxAuditSetup(t)
	inboundService := &InboundService{}
	inbound, _, err := inboundService.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatalf("create sing-box inbound: %v", err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	clients, err := inboundService.GetClients(inbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("read sing-box account: %d clients, %v", len(clients), err)
	}
	email := clients[0].Email
	newExpiry := time.Now().Add(24 * time.Hour).UnixMilli()
	if _, _, err := inboundService.updateSingboxAccount(inbound, email, func(account *model.Account) {
		account.TotalGB = 1024
	}); err != nil {
		t.Fatalf("set initial quota: %v", err)
	}

	ticket := BulkTicket{Active: true, Charges: []BulkCharge{{
		Email: email, SetTotal: true, NewTotal: 2048, ForceExpiry: true, ExpiryTime: newExpiry,
	}}}
	if err := (&ResellerService{}).ApplyBulkCharges(ticket); err != nil {
		t.Fatalf("apply sing-box reseller charge: %v", err)
	}

	account, err := (&AccountService{}).GetAccountByEmail(email)
	if err != nil || account == nil {
		t.Fatalf("load account policy: %+v, %v", account, err)
	}
	if account.TotalGB != 2048 || account.ExpiryTime != newExpiry {
		t.Fatalf("account policy = quota %d expiry %d, want 2048 and %d", account.TotalGB, account.ExpiryTime, newExpiry)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil {
		t.Fatalf("load traffic policy: %v", err)
	}
	if traffic.Total != 2048 || traffic.ExpiryTime != newExpiry {
		t.Fatalf("traffic policy = quota %d expiry %d, want 2048 and %d", traffic.Total, traffic.ExpiryTime, newExpiry)
	}

	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		_, err := (&AccountService{}).ProjectAccount(tx, account.Id)
		return err
	}); err != nil {
		t.Fatalf("re-project account after bulk charge: %v", err)
	}
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil {
		t.Fatalf("reload traffic policy: %v", err)
	}
	if traffic.Total != 2048 || traffic.ExpiryTime != newExpiry {
		t.Errorf("later account projection reverted the bulk policy: quota %d expiry %d", traffic.Total, traffic.ExpiryTime)
	}
}
