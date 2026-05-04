package earnburn

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
	"github.com/theluisbolivar/fidel-quick/internal/phone"
)

// Module implements loyalty.Module for the earn-burn system.
type Module struct {
	service *Service
	api     *APIHandler
}

func NewModule(service *Service, api *APIHandler) *Module {
	return &Module{service: service, api: api}
}

func (m *Module) Name() string { return "earn_burn" }

func (m *Module) Menus() map[string][]loyalty.MenuDefinition {
	return map[string][]loyalty.MenuDefinition{
		"client":       ClientMenus(),
		"collaborator": CollaboratorMenus(),
	}
}

func (m *Module) FlowDefinitions() map[string]loyalty.FlowDefinition {
	return FlowDefs()
}

func (m *Module) Prefixes() []string { return []string{"reward:"} }

func (m *Module) SelectionFlow(prefix string) (string, string) {
	return "request_redemption", "reward_id"
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	if m.api != nil {
		m.api.RegisterRoutes(rg)
	}
}

func (m *Module) HandleCommand(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	switch cmd.ID {
	case "check_points":
		return m.handleCheckPoints(ctx, cmd)
	case "list_all_rewards":
		return m.handleListRewards(ctx, cmd)
	case "redeem_rewards":
		return m.handleRedeemRewards(ctx, cmd)
	case "request_otp":
		return m.handleRequestOTP(ctx, cmd)
	case "request_redemption":
		return m.handleRequestRedemption(ctx, cmd)
	case "confirm_redemption":
		return m.handleConfirmRedemption(ctx, cmd)
	case "load_points_request":
		return m.handleLoadPointsRequest(ctx, cmd)
	case "load_points_process":
		return m.handleLoadPointsProcess(ctx, cmd)
	case "add_points":
		return m.handleAddPoints(ctx, cmd)
	case "list_points":
		return m.handleListPoints(ctx, cmd)
	case "update_points":
		return m.handleUpdatePoints(ctx, cmd)
	case "submit_feedback":
		return m.handleSubmitFeedback(ctx, cmd)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd.ID)
	}
}

func (m *Module) handleCheckPoints(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	balance, err := m.service.CheckBalance(ctx, cmd.UserContext.UserID, program.CustomerSisfiID)
	if err != nil {
		return nil, err
	}

	txs, err := m.service.ListTransactions(ctx, cmd.UserContext.UserID, program.CustomerSisfiID, 5)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Tienes *%d puntos*.", balance)
	if len(txs) > 0 {
		msg += "\n\nUltimos movimientos:"
		for _, tx := range txs {
			sign := "+"
			if tx.Amount < 0 {
				sign = ""
			}
			msg += fmt.Sprintf("\n%s%d pts - %s (%s)", sign, tx.Amount, txTypeLabel(tx.Type), tx.CreatedAt.Format("02 Jan"))
		}
	}

	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleListRewards(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	balance, err := m.service.CheckBalance(ctx, cmd.UserContext.UserID, program.CustomerSisfiID)
	if err != nil {
		return nil, err
	}

	rewards, err := m.service.ListRewards(ctx, cmd.UserContext.CustomerID, program.CustomerSisfiID, 999999)
	if err != nil {
		return nil, err
	}

	if len(rewards) == 0 {
		return &loyalty.CommandResult{Message: "No hay recompensas disponibles."}, nil
	}

	msg := fmt.Sprintf("Tienes *%d puntos*.\n\nRecompensas disponibles:", balance)
	for _, rw := range rewards {
		status := "disponible"
		if rw.PointsCost > balance {
			status = fmt.Sprintf("te faltan %d pts", rw.PointsCost-balance)
		}
		msg += fmt.Sprintf("\n- *%s* — %d pts (%s)", rw.Name, rw.PointsCost, status)
	}
	msg += "\n\nSigue acumulando puntos para desbloquear mas premios."

	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleRedeemRewards(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	program, err := m.service.GetProgram(ctx, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	balance, err := m.service.CheckBalance(ctx, cmd.UserContext.UserID, program.CustomerSisfiID)
	if err != nil {
		return nil, err
	}

	// Only show rewards the client can afford
	rewards, err := m.service.ListRewards(ctx, cmd.UserContext.CustomerID, program.CustomerSisfiID, balance)
	if err != nil {
		return nil, err
	}

	if len(rewards) == 0 {
		return &loyalty.CommandResult{
			Message: fmt.Sprintf("No tienes puntos suficientes para canjear.\nTu balance: *%d puntos*.\n\nSigue acumulando para desbloquear premios.", balance),
		}, nil
	}

	msg := fmt.Sprintf("Tienes *%d puntos*. Selecciona una recompensa:", balance)

	var options []loyalty.CommandOption
	for _, rw := range rewards {
		options = append(options, loyalty.CommandOption{
			ID:          "reward:" + rw.ID,
			Title:       rw.Name,
			Description: fmt.Sprintf("%d pts", rw.PointsCost),
		})
	}

	return &loyalty.CommandResult{
		Message:    fmt.Sprintf(msg, balance),
		Options:    options,
		ListHeader: "Canjear",
	}, nil
}

func (m *Module) handleRequestOTP(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code, err := m.service.RequestIdentityOTP(ctx, cmd.UserContext.UserID, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Tu codigo de identificacion: *%s*\nValido por 15 minutos.\nMuestraselo al colaborador.", code)
	return &loyalty.CommandResult{Message: msg}, nil
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

	rd, code, err := m.service.RequestRedemption(ctx, RedemptionReq{
		ClientID:        cmd.UserContext.UserID,
		CustomerSisfiID: program.CustomerSisfiID,
		RewardID:        rewardID,
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
	rewardName := "Recompensa"
	if reward != nil {
		rewardName = reward.Name
	}

	msg := fmt.Sprintf("Canje confirmado.\nEntrega: *%s*", rewardName)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleLoadPointsRequest(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code, err := m.service.RequestLoadPointsCode(ctx, cmd.UserContext.UserID, cmd.UserContext.CustomerID)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Tu codigo de carga: *%s*\nValido por 15 minutos.\nDaselo al colaborador junto con tu ticket.", code)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleLoadPointsProcess(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	code := cmd.Data["code"]
	otpData, err := m.service.ValidateLoadPointsCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Prevent self-accrual
	clientPhone, err := m.service.GetClientPhone(ctx, otpData.ClientID)
	if err != nil {
		return nil, fmt.Errorf("get client phone: %w", err)
	}
	if phone.SameNumber(clientPhone, cmd.UserContext.Phone) {
		return nil, fmt.Errorf("no puedes acreditar puntos a ti mismo")
	}

	// Photo URL from AI processing (amount extracted by flow engine)
	photoURL := cmd.Data["photo"]
	amountStr := cmd.Data["amount"]
	amount, _ := strconv.ParseFloat(amountStr, 64)

	program, err := m.service.GetProgram(ctx, otpData.CustomerID)
	if err != nil {
		return nil, err
	}

	tx, err := m.service.AddPoints(ctx, AddPointsReq{
		ClientID:        otpData.ClientID,
		CustomerSisfiID: program.CustomerSisfiID,
		CollaboratorID:  cmd.UserContext.UserID,
		Amount:          amount,
		InvoiceURL:      photoURL,
	})
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Carga exitosa. *%d puntos* agregados.\nBalance: %d puntos.", tx.Amount, tx.BalanceAfter)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleAddPoints(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	otp := cmd.Data["otp"]
	otpData, err := m.service.ValidateIdentityOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	// Prevent self-accrual
	clientPhone, err := m.service.GetClientPhone(ctx, otpData.ClientID)
	if err != nil {
		return nil, fmt.Errorf("get client phone: %w", err)
	}
	if phone.SameNumber(clientPhone, cmd.UserContext.Phone) {
		return nil, fmt.Errorf("no puedes acreditar puntos a ti mismo")
	}

	photoURL := cmd.Data["photo"]
	amountStr := cmd.Data["amount"]
	amount, _ := strconv.ParseFloat(amountStr, 64)

	program, err := m.service.GetProgram(ctx, otpData.CustomerID)
	if err != nil {
		return nil, err
	}

	tx, err := m.service.AddPoints(ctx, AddPointsReq{
		ClientID:        otpData.ClientID,
		CustomerSisfiID: program.CustomerSisfiID,
		CollaboratorID:  cmd.UserContext.UserID,
		Amount:          amount,
		InvoiceURL:      photoURL,
	})
	if err != nil {
		return nil, err
	}

	corrWindow := ""
	if tx.CorrectableUntil != nil {
		remaining := time.Until(*tx.CorrectableUntil).Truncate(time.Minute)
		corrWindow = fmt.Sprintf("\nCorreccion disponible por %s.", remaining)
	}

	msg := fmt.Sprintf("Se agrego *%d punto(s)*.\nBalance: %d puntos.%s", tx.Amount, tx.BalanceAfter, corrWindow)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleListPoints(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
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
	balance, err := m.service.CheckBalance(ctx, otpData.ClientID, program.CustomerSisfiID)
	if err != nil {
		return nil, err
	}

	txs, err := m.service.ListTransactions(ctx, otpData.ClientID, program.CustomerSisfiID, 10)
	if err != nil {
		return nil, err
	}

	displayName := clientName
	if displayName == "" {
		displayName = "Cliente"
	}

	msg := fmt.Sprintf("Cliente: *%s*\nBalance: *%d puntos*", displayName, balance)
	if len(txs) > 0 {
		msg += "\n\nHistorial:"
		for _, tx := range txs {
			sign := "+"
			if tx.Amount < 0 {
				sign = ""
			}
			msg += fmt.Sprintf("\n%s%d pts - %s (%s)", sign, tx.Amount, txTypeLabel(tx.Type), tx.CreatedAt.Format("02 Jan"))
		}
	}

	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleUpdatePoints(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	otp := cmd.Data["otp"]
	_, err := m.service.ValidateIdentityOTP(ctx, otp)
	if err != nil {
		return nil, err
	}

	txID := cmd.Data["tx_id"]
	newAmountStr := cmd.Data["new_amount"]
	newAmount, _ := strconv.Atoi(newAmountStr)
	evidence := cmd.Data["evidence"]
	reason := cmd.Data["reason"]

	tx, err := m.service.UpdatePoints(ctx, UpdatePointsReq{
		TransactionID:         txID,
		CollaboratorID:        cmd.UserContext.UserID,
		NewAmount:             newAmount,
		CorrectionReason:      reason,
		CorrectionEvidenceURL: evidence,
	})
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Correccion aplicada.\nAjuste: %+d puntos.\nBalance: %d puntos.", tx.Amount, tx.BalanceAfter)
	return &loyalty.CommandResult{Message: msg}, nil
}

func (m *Module) handleSubmitFeedback(ctx context.Context, cmd loyalty.Command) (*loyalty.CommandResult, error) {
	message := cmd.Data["message"]
	if err := m.service.SubmitFeedback(ctx, cmd.UserContext.UserID, cmd.UserContext.CustomerID, message); err != nil {
		return nil, err
	}
	return &loyalty.CommandResult{Message: "Gracias por tu feedback."}, nil
}

func txTypeLabel(t string) string {
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
