package pushcard

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

func newTestModule(t *testing.T) (*Module, *fakeRepo, *fakeCache) {
	t.Helper()
	repo := newFakeRepo()
	cache := &fakeCache{}
	svc := NewService(repo, cache, slog.New(slog.NewTextHandler(io.Discard, nil)))
	api := NewAPIHandler(svc)
	return NewModule(svc, api), repo, cache
}

func TestModule_Name(t *testing.T) {
	m, _, _ := newTestModule(t)
	if m.Name() != "pushcard" {
		t.Fatalf("want pushcard, got %s", m.Name())
	}
}

func TestModule_Menus_HasClientAndCollaborator(t *testing.T) {
	m, _, _ := newTestModule(t)
	menus := m.Menus()
	if _, ok := menus["client"]; !ok {
		t.Fatalf("missing client menus")
	}
	if _, ok := menus["collaborator"]; !ok {
		t.Fatalf("missing collaborator menus")
	}
}

func TestModule_Prefixes_Empty(t *testing.T) {
	m, _, _ := newTestModule(t)
	if len(m.Prefixes()) != 0 {
		t.Fatalf("pushcard reward is fixed in config; expected empty prefixes, got %v", m.Prefixes())
	}
}

func TestModule_HandleCommand_Unknown(t *testing.T) {
	m, _, _ := newTestModule(t)
	_, err := m.HandleCommand(context.Background(), loyalty.Command{ID: "no_such_command"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

func TestModule_PcCheckCard_NoOpenCard(t *testing.T) {
	m, repo, _ := newTestModule(t)
	repo.configs["cs-1"] = &Config{CustomerSisfiID: "cs-1", CustomerID: "cust-1", Name: "Café", CardSlots: 5, Active: true}

	res, err := m.HandleCommand(context.Background(), loyalty.Command{
		ID:          "pc_check_card",
		UserContext: loyalty.UserContext{CustomerID: "cust-1", UserID: "client-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Message, "Aún no tenés sellos") {
		t.Fatalf("expected first-stamp message, got: %s", res.Message)
	}
}

func TestModule_PcCheckCard_ShowsProgress(t *testing.T) {
	m, repo, _ := newTestModule(t)
	repo.configs["cs-1"] = &Config{CustomerSisfiID: "cs-1", CustomerID: "cust-1", Name: "Café", CardSlots: 5, Active: true}

	// Add 2 stamps via service
	for i := 0; i < 2; i++ {
		if _, err := m.service.AddStamp(context.Background(), AddStampReq{
			CustomerSisfiID: "cs-1", ClientID: "client-1", CollaboratorID: "k",
		}); err != nil {
			t.Fatal(err)
		}
	}

	res, _ := m.HandleCommand(context.Background(), loyalty.Command{
		ID:          "pc_check_card",
		UserContext: loyalty.UserContext{CustomerID: "cust-1", UserID: "client-1"},
	})
	if !strings.Contains(res.Message, "●●○○○") {
		t.Fatalf("expected ●●○○○ in: %s", res.Message)
	}
	if !strings.Contains(res.Message, "2 / 5") {
		t.Fatalf("expected 2 / 5 in: %s", res.Message)
	}
}

func TestModule_PcRedeem_NotComplete(t *testing.T) {
	m, repo, _ := newTestModule(t)
	repo.configs["cs-1"] = &Config{CustomerSisfiID: "cs-1", CustomerID: "cust-1", Name: "Café", CardSlots: 5, Active: true}
	for i := 0; i < 3; i++ {
		_, _ = m.service.AddStamp(context.Background(), AddStampReq{
			CustomerSisfiID: "cs-1", ClientID: "client-1", CollaboratorID: "k",
		})
	}

	res, _ := m.HandleCommand(context.Background(), loyalty.Command{
		ID:          "pc_redeem",
		UserContext: loyalty.UserContext{CustomerID: "cust-1", UserID: "client-1"},
	})
	if !strings.Contains(res.Message, "Te faltan 2 sellos") {
		t.Fatalf("expected 'Te faltan 2 sellos' in: %s", res.Message)
	}
}

func TestModule_PcRedeem_Complete_GeneratesCode(t *testing.T) {
	m, repo, cache := newTestModule(t)
	repo.configs["cs-1"] = &Config{CustomerSisfiID: "cs-1", CustomerID: "cust-1", Name: "Café", CardSlots: 2, Active: true}
	for i := 0; i < 2; i++ {
		_, _ = m.service.AddStamp(context.Background(), AddStampReq{
			CustomerSisfiID: "cs-1", ClientID: "client-1", CollaboratorID: "k",
		})
	}

	res, err := m.HandleCommand(context.Background(), loyalty.Command{
		ID:          "pc_redeem",
		UserContext: loyalty.UserContext{CustomerID: "cust-1", UserID: "client-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Message, "código de canje") {
		t.Fatalf("expected redemption code in: %s", res.Message)
	}
	if len(cache.m) != 1 {
		t.Fatalf("expected 1 cached code, got %d", len(cache.m))
	}
}

func TestModule_PcConfirmRedemption_Roundtrip(t *testing.T) {
	m, repo, _ := newTestModule(t)
	repo.configs["cs-1"] = &Config{CustomerSisfiID: "cs-1", CustomerID: "cust-1", Name: "Café", CardSlots: 2, Active: true}
	for i := 0; i < 2; i++ {
		_, _ = m.service.AddStamp(context.Background(), AddStampReq{
			CustomerSisfiID: "cs-1", ClientID: "client-1", CollaboratorID: "k",
		})
	}
	code, err := m.service.RequestRedemption(context.Background(), "cs-1", "client-1", "cust-1", "Café")
	if err != nil {
		t.Fatal(err)
	}

	res, err := m.HandleCommand(context.Background(), loyalty.Command{
		ID:          "pc_confirm_redemption",
		UserContext: loyalty.UserContext{CustomerID: "cust-1", UserID: "collab-1"},
		Data:        map[string]string{"code": code},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Message, "Canje confirmado") {
		t.Fatalf("expected confirmation, got: %s", res.Message)
	}
}

func TestModule_PcConfirmRedemption_BadCode(t *testing.T) {
	m, _, _ := newTestModule(t)
	res, _ := m.HandleCommand(context.Background(), loyalty.Command{
		ID:   "pc_confirm_redemption",
		Data: map[string]string{"code": "999999"},
	})
	if !strings.Contains(res.Message, "inválido") && !strings.Contains(res.Message, "expirado") {
		t.Fatalf("expected invalid/expired message, got: %s", res.Message)
	}
}

func TestModule_RegistryRegistration(t *testing.T) {
	m, _, _ := newTestModule(t)

	r := loyalty.NewRegistry()
	r.Register(m)

	clientMenus := r.AllMenus("client")
	if len(clientMenus) == 0 {
		t.Fatalf("expected pushcard client menus to be registered")
	}
}
