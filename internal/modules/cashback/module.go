package cashback

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

// Module implements loyalty.Module for the cashback system.
type Module struct {
	service *Service
	api     *APIHandler
}

func NewModule(service *Service, api *APIHandler) *Module {
	return &Module{service: service, api: api}
}

func (m *Module) Name() string { return "cashback" }

func (m *Module) Menus() map[string][]loyalty.MenuDefinition {
	return map[string][]loyalty.MenuDefinition{
		"client":       ClientMenus(),
		"collaborator": CollaboratorMenus(),
	}
}

func (m *Module) FlowDefinitions() map[string]loyalty.FlowDefinition {
	return FlowDefs()
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	if m.api != nil {
		m.api.RegisterRoutes(rg)
	}
}

func (m *Module) HandleCommand(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	switch cmd.ID {
	case "cb_check_balance":
		return m.handleCheckBalance(ctx, cmd)
	case "cb_list_rewards":
		return m.handleListRewards(ctx, cmd)
	case "cb_redeem":
		return m.handleRedeem(ctx, cmd)
	case "cb_request_redemption":
		return m.handleRequestRedemption(ctx, cmd)
	case "cb_confirm_redemption":
		return m.handleConfirmRedemption(ctx, cmd)
	case "cb_load_request":
		return m.handleLoadRequest(ctx, cmd)
	case "cb_load_process":
		return m.handleLoadProcess(ctx, cmd)
	case "cb_add_cashback":
		return m.handleAddCashback(ctx, cmd)
	case "cb_list_balance":
		return m.handleListBalance(ctx, cmd)
	case "cb_update_cashback":
		return m.handleUpdateCashback(ctx, cmd)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd.ID)
	}
}

func (m *Module) handleCheckBalance(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	balance, err := m.service.CheckBalance(ctx, cmd.UserContext.UserID, program.ID)
	if err != nil {
		return nil, err
	}

	txs, err := m.service.ListTransactions(ctx, cmd.UserContext.UserID, program.ID, 5)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Tienes *$%.2f MXN* de cashback.", balance)
	if len(txs) > 0 {
		msg += "\n\nUltimos movimientos:"
		for _, tx := range txs {
			sign := "+"
			if tx.Amount < 0 {
				sign = ""
			}
			msg += fmt.Sprintf("\n%s$%.2f - %s (%s)", sign, tx.Amount, cbTxTypeLabel(tx.Type), tx.CreatedAt.Format("02 Jan"))
		}
	}

	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleListRewards(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	balance, err := m.service.CheckBalance(ctx, cmd.UserContext.UserID, program.ID)
	if err != nil {
		return nil, err
	}

	rewards, err := m.service.ListRewards(ctx, cmd.UserContext.CustomerID, program.ID, 999999)
	if err != nil {
		return nil, err
	}

	if len(rewards) == 0 {
		return &loyalty.CommandResult{Message: "No hay beneficios disponibles."}, nil
	}

	msg := fmt.Sprintf("Tienes *$%.2f MXN*.\n\nBeneficios disponibles:", balance)
	for _, rw := range rewards {
		status := "disponible"
		if rw.Cost > balance {
			status = fmt.Sprintf("te faltan $%.2f", rw.Cost-balance)
		}
		msg += fmt.Sprintf("\n- *%s* — $%.2f (%s)", rw.Name, rw.Cost, status)
	}
	msg += "\n\nSigue acumulando cashback para desbloquear mas beneficios."

	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleRedeem(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	balance, err := m.service.CheckBalance(ctx, cmd.UserContext.UserID, program.ID)
	if err != nil {
		return nil, err
	}

	// Only show rewards the client can afford
	rewards, err := m.service.ListRewards(ctx, cmd.UserContext.CustomerID, program.ID, balance)
	if err != nil {
		return nil, err
	}

	if len(rewards) == 0 {
		return &loyalty.CommandResult{
			Message: fmt.Sprintf("No tienes saldo suficiente para canjear.\nTu saldo: *$%.2f MXN*.\n\nSigue acumulando para desbloquear beneficios.", balance),
		}, nil
	}

	var options []loyalty.CommandOption
	for _, rw := range rewards {
		options = append(options, loyalty.CommandOption{
			ID:          "benefit:" + rw.ID,
			Title:       rw.Name,
			Description: fmt.Sprintf("$%.2f", rw.Cost),
		})
	}

	return &loyalty.CommandResult{
		Message:    fmt.Sprintf("Tienes *$%.2f MXN*. Selecciona un beneficio:", balance),
		Options:    options,
		ListHeader: "Canjear",
	}, nil
}

func (m *Module) handleRequestRedemption(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	confirm := cmd.Data["confirm"]
	if strings.ToLower(confirm) == "no" {
		return &loyalty.CommandResult{Message: "Canje cancelado."}, nil
	}

	rewardID := cmd.Data["reward_id"]
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	rd, code, err := m.service.RequestRedemption(ctx, CashbackRedemptionReq{
		ClientID:  cmd.UserContext.UserID,
		ProgramID: program.ID,
		RewardID:  rewardID,
	})
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Tu codigo de canje: *%s*\nValido por 1 hora.\nMuestraselo al colaborador.", code)
	_ = rd
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleConfirmRedemption(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code := cmd.Data["code"]
	rd, err := m.service.ConfirmRedemption(ctx, code, cmd.UserContext.UserID)
	if err != nil {
		return nil, err
	}

	reward, _ := m.service.GetReward(ctx, rd.RewardID)
	rewardName := "Beneficio"
	if reward != nil {
		rewardName = reward.Name
	}

	msg := fmt.Sprintf("Canje confirmado.\nEntrega: *%s*", rewardName)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleLoadRequest(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code, err := m.service.RequestLoadCode(ctx, cmd.UserContext.UserID, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Tu codigo de carga: *%s*\nValido por 15 minutos.\nDaselo al colaborador junto con tu ticket.", code)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleLoadProcess(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code := cmd.Data["code"]
	otpData, err := m.service.ValidateLoadCode(ctx, code)
	if err != nil {
		return nil, err
	}

	photoURL := cmd.Data["photo"]
	amountStr := cmd.Data["amount"]
	amount, _ := strconv.ParseFloat(amountStr, 64)

	program, err := m.service.GetProgram(ctx, otpData.CustomerID)
	if err != nil {
		return nil, err
	}

	tx, err := m.service.AddCashback(ctx, AddCashbackReq{
		ClientID:       otpData.ClientID,
		ProgramID:      program.ID,
		CollaboratorID: cmd.UserContext.UserID,
		Amount:         amount,
		InvoiceURL:     photoURL,
	})
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Carga exitosa. *$%.2f MXN* de cashback agregados.\nSaldo: $%.2f MXN.", tx.Amount, tx.BalanceAfter)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleAddCashback(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	otp := cmd.Data["otp"]
	otpData, err := m.service.ValidateIdentityOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	photoURL := cmd.Data["photo"]
	amountStr := cmd.Data["amount"]
	amount, _ := strconv.ParseFloat(amountStr, 64)

	program, err := m.service.GetProgram(ctx, otpData.CustomerID)
	if err != nil {
		return nil, err
	}

	tx, err := m.service.AddCashback(ctx, AddCashbackReq{
		ClientID:       otpData.ClientID,
		ProgramID:      program.ID,
		CollaboratorID: cmd.UserContext.UserID,
		Amount:         amount,
		InvoiceURL:     photoURL,
	})
	if err != nil {
		return nil, err
	}

	corrWindow := ""
	if tx.CorrectableUntil != nil {
		remaining := time.Until(*tx.CorrectableUntil).Truncate(time.Minute)
		corrWindow = fmt.Sprintf("\nCorreccion disponible por %s.", remaining)
	}

	msg := fmt.Sprintf("Se agrego *$%.2f MXN* de cashback.\nSaldo: $%.2f MXN.%s", tx.Amount, tx.BalanceAfter, corrWindow)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleListBalance(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	otp := cmd.Data["otp"]
	otpData, err := m.service.ValidateIdentityOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	program, err := m.service.GetProgram(ctx, otpData.CustomerID)
	if err != nil {
		return nil, err
	}

	clientName, _ := m.service.GetClientName(ctx, otpData.ClientID)
	balance, err := m.service.CheckBalance(ctx, otpData.ClientID, program.ID)
	if err != nil {
		return nil, err
	}

	txs, err := m.service.ListTransactions(ctx, otpData.ClientID, program.ID, 10)
	if err != nil {
		return nil, err
	}

	displayName := clientName
	if displayName == "" {
		displayName = "Cliente"
	}

	msg := fmt.Sprintf("Cliente: *%s*\nSaldo: *$%.2f MXN*", displayName, balance)
	if len(txs) > 0 {
		msg += "\n\nHistorial:"
		for _, tx := range txs {
			sign := "+"
			if tx.Amount < 0 {
				sign = ""
			}
			msg += fmt.Sprintf("\n%s$%.2f - %s (%s)", sign, tx.Amount, cbTxTypeLabel(tx.Type), tx.CreatedAt.Format("02 Jan"))
		}
	}

	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleUpdateCashback(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	otp := cmd.Data["otp"]
	_, err := m.service.ValidateIdentityOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	txID := cmd.Data["tx_id"]
	newAmountStr := cmd.Data["new_amount"]
	newAmount, _ := strconv.ParseFloat(newAmountStr, 64)
	evidence := cmd.Data["evidence"]
	reason := cmd.Data["reason"]

	tx, err := m.service.UpdateCashback(ctx, UpdateCashbackReq{
		TransactionID:         txID,
		CollaboratorID:        cmd.UserContext.UserID,
		NewPurchaseAmount:     newAmount,
		CorrectionReason:      reason,
		CorrectionEvidenceURL: evidence,
	})
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Correccion aplicada.\nAjuste: %+.2f MXN.\nSaldo: $%.2f MXN.", tx.Amount, tx.BalanceAfter)
	return &loyalty.CommandResult{Message: msg}, nil
}

func cbTxTypeLabel(t string) string {
	switch t {
	case "earn":
		return "Carga"
	case "burn":
		return "Canje"
	case "adjustment":
		return "Correccion"
	default:
		return t
	}
}
