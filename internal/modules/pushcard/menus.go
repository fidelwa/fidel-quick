package pushcard

import (
	"fmt"

	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
)

// ClientMenus returns the menu options for clients.
func ClientMenus() []loyalty.MenuDefinition {
	return []loyalty.MenuDefinition{
		{ID: "pc_check_card", Title: "Mi tarjeta de sellos", Description: "Ver progreso de mi tarjeta", Role: "client"},
		{ID: "pc_redeem", Title: "Canjear tarjeta", Description: "Canjear tarjeta completada", Role: "client"},
	}
}

// CollaboratorMenus returns the collaborator menu options.
func CollaboratorMenus() []loyalty.MenuDefinition {
	return []loyalty.MenuDefinition{
		{ID: "pc_add_stamp", Title: "Sumar sello", Description: "Agregar un sello a un cliente", Role: "collaborator"},
		{ID: "pc_undo_stamp", Title: "Deshacer último sello", Description: "Corrección dentro de 2 horas", Role: "collaborator"},
		{ID: "pc_confirm_redemption", Title: "Confirmar canje de tarjeta", Description: "Validar canje del cliente", Role: "collaborator"},
	}
}

// FlowDefs returns the multi-step flows for pushcard.
func FlowDefs() map[string]loyalty.FlowDefinition {
	return map[string]loyalty.FlowDefinition{
		"pc_add_stamp": {
			CommandID: "pc_add_stamp",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_phone", Prompt: "Escribe el teléfono del cliente:", Key: "client_phone", Validate: validatePhone},
			},
		},
		"pc_confirm_redemption": {
			CommandID: "pc_confirm_redemption",
			Steps: []loyalty.StepDefinition{
				{ID: "ask_code", Prompt: "Escribe el código de canje del cliente (6 dígitos):", Key: "code", Validate: validateCodeFormat},
			},
		},
	}
}

func validateCodeFormat(input string) error {
	if len(input) != 6 {
		return fmt.Errorf("el código debe tener 6 dígitos")
	}
	for _, c := range input {
		if c < '0' || c > '9' {
			return fmt.Errorf("el código solo debe contener dígitos")
		}
	}
	return nil
}

func validatePhone(input string) error {
	digits := 0
	for _, c := range input {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	if digits < 10 {
		return fmt.Errorf("escribe un teléfono válido (mínimo 10 dígitos)")
	}
	return nil
}
