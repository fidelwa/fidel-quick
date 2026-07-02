package earnburn

import "math"

// MetricsRaw contiene los contadores crudos agregados por SQL sobre las tablas
// operacionales de un customer (earnburn + cashback). ComputeMetrics deriva de
// aquí las métricas T1 del dashboard. Separar la agregación cruda del cálculo
// permite testear la lógica de derivación con datos sembrados, sin base de datos.
type MetricsRaw struct {
	// Clientes
	RegisteredClients  int // total de clientes registrados del customer
	ActiveClients      int // clientes con actividad (transacción) en la ventana reciente
	ReactivatedClients int // clientes activos hoy que estuvieron inactivos antes de la ventana

	// Participación / compras
	ParticipatingClients int     // clientes con al menos una transacción de compra (earn)
	PurchaseCount        int     // # de transacciones de compra (earn) en ambos sistemas
	TotalSpend           float64 // gasto total acumulado por clientes participantes (MXN)

	// Beneficios
	BenefitsGenerated float64 // valor de beneficios generados (puntos valorizados + cashback)
	BenefitsRedeemed  float64 // valor de beneficios efectivamente redimidos (canjes confirmados)
	BenefitsCost      float64 // costo de los beneficios entregados al cliente (= redimidos)
	OutstandingValue  float64 // valor de saldos sin redimir (pasivo / ganancia potencial retenida)

	// Redenciones
	TotalRedemptions     int // canjes totales creados
	ConfirmedRedemptions int // canjes confirmados
}

// CustomerMetrics son las métricas T1 expuestas por el endpoint del dashboard.
// Todas las tasas se expresan en fracción [0,1]; el frontend las formatea a %.
type CustomerMetrics struct {
	RegisteredClients  int     `json:"registered_clients"`
	ActiveClients      int     `json:"active_clients"`
	ReactivatedClients int     `json:"reactivated_clients"`
	ParticipationRate  float64 `json:"participation_rate"` // participantes / registrados
	PurchaseFrequency  float64 `json:"purchase_frequency"` // compras promedio por cliente participante
	AverageTicket      float64 `json:"average_ticket"`     // gasto por compra
	SpendPerClient     float64 `json:"spend_per_client"`   // gasto por cliente participante
	TotalSpend         float64 `json:"total_spend"`        // gasto total de participantes
	BenefitsGenerated  float64 `json:"benefits_generated"` // valor de beneficios generados
	BenefitsRedeemed   float64 `json:"benefits_redeemed"`  // valor de beneficios redimidos
	BenefitsCost       float64 `json:"benefits_cost"`      // costo de beneficios entregados
	PotentialGain      float64 `json:"potential_gain"`     // saldos retenidos sin redimir
	RedemptionRate     float64 `json:"redemption_rate"`    // canjes confirmados / totales
}

// round2 redondea a 2 decimales para montos y tasas monetarias legibles.
func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

// safeDiv divide protegiendo contra denominador cero.
func safeDiv(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

// ComputeMetrics deriva las métricas T1 a partir de los contadores crudos.
// Función pura: sin efectos secundarios ni acceso a I/O, testeable en aislamiento.
func ComputeMetrics(raw MetricsRaw) CustomerMetrics {
	return CustomerMetrics{
		RegisteredClients:  raw.RegisteredClients,
		ActiveClients:      raw.ActiveClients,
		ReactivatedClients: raw.ReactivatedClients,
		ParticipationRate: round2(
			safeDiv(float64(raw.ParticipatingClients), float64(raw.RegisteredClients)),
		),
		PurchaseFrequency: round2(
			safeDiv(float64(raw.PurchaseCount), float64(raw.ParticipatingClients)),
		),
		AverageTicket: round2(
			safeDiv(raw.TotalSpend, float64(raw.PurchaseCount)),
		),
		SpendPerClient: round2(
			safeDiv(raw.TotalSpend, float64(raw.ParticipatingClients)),
		),
		TotalSpend:        round2(raw.TotalSpend),
		BenefitsGenerated: round2(raw.BenefitsGenerated),
		BenefitsRedeemed:  round2(raw.BenefitsRedeemed),
		BenefitsCost:      round2(raw.BenefitsCost),
		PotentialGain:     round2(raw.OutstandingValue),
		RedemptionRate: round2(
			safeDiv(float64(raw.ConfirmedRedemptions), float64(raw.TotalRedemptions)),
		),
	}
}
