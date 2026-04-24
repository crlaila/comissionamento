package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"comissionamento/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoalRepository interface {
	GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Goal, error)
	ListByPeriod(ctx context.Context, periodID int64) ([]*model.Goal, error)
}

type PeriodRepository interface {
	GetByID(ctx context.Context, id int64) (*model.Period, error)
	List(ctx context.Context) ([]*model.Period, error)
}

type StatementRepository interface {
	Create(ctx context.Context, statement *model.Statement) error
	GetByRepAndPeriod(ctx context.Context, repID, periodID int64) (*model.Statement, error)
	ListByPeriod(ctx context.Context, periodID int64) ([]*model.Statement, error)
	UpdateStatus(ctx context.Context, id int64, status model.StatementStatus, approverID *int64, reason *string) error
}

type CommissionEventRepository interface {
	CountByRepAndType(ctx context.Context, repID int64, eventType model.EventType, startDate, endDate interface{}) (int, error)
	ListByRepAndPeriod(ctx context.Context, repID int64, startDate, endDate interface{}) ([]*model.MemberEvent, error)
}

type CommissionService interface {
	CalculateForPeriod(ctx context.Context, periodID int64) error
	GetRepDashboard(ctx context.Context, repID int64) (*model.RepDashboard, error)
	GetTeamDashboard(ctx context.Context, managerID int64) (*model.TeamDashboard, error)
	GenerateStatements(ctx context.Context, periodID int64) error
	ApproveStatement(ctx context.Context, statementID int64, approverID int64) error
}

type commissionService struct {
	goalRepository      GoalRepository
	periodRepository    PeriodRepository
	userRepository      UserRepository
	statementRepository StatementRepository
	eventRepository     CommissionEventRepository
	pool                *pgxpool.Pool
}

func NewCommissionService(
	goalRepository GoalRepository,
	periodRepository PeriodRepository,
	userRepository UserRepository,
	statementRepository StatementRepository,
	eventRepository CommissionEventRepository,
	pool *pgxpool.Pool,
) CommissionService {
	return &commissionService{
		goalRepository:      goalRepository,
		periodRepository:    periodRepository,
		userRepository:      userRepository,
		statementRepository: statementRepository,
		eventRepository:     eventRepository,
		pool:                pool,
	}
}

// CalculateForPeriod computes commissions for all reps in a period
func (s *commissionService) CalculateForPeriod(ctx context.Context, periodID int64) error {
	startTime := time.Now()

	// Get period to access date range
	period, err := s.periodRepository.GetByID(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to get period: %w", err)
	}
	if period == nil {
		return fmt.Errorf("period not found: %d", periodID)
	}

	// Get all goals for this period to find all reps with goals
	goals, err := s.goalRepository.ListByPeriod(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to list goals by period: %w", err)
	}

	if len(goals) == 0 {
		slog.Info("no goals found for period", "period_id", periodID)
		return nil
	}

	repsProcessed := 0
	totalCommissionAmount := int64(0)

	// Process each rep's commission
	for _, goal := range goals {
		statement, err := s.calculateRepCommission(ctx, goal, period)
		if err != nil {
			slog.Error("failed to calculate rep commission",
				"rep_id", goal.RepID,
				"period_id", periodID,
				"error", err,
			)
			continue
		}

		// Check if statement already exists, if so skip creation
		existing, err := s.statementRepository.GetByRepAndPeriod(ctx, goal.RepID, periodID)
		if err != nil {
			slog.Error("failed to check existing statement",
				"rep_id", goal.RepID,
				"period_id", periodID,
				"error", err,
			)
			continue
		}

		if existing == nil {
			if err := s.statementRepository.Create(ctx, statement); err != nil {
				slog.Error("failed to create statement",
					"rep_id", goal.RepID,
					"period_id", periodID,
					"error", err,
				)
				continue
			}
		}

		repsProcessed++
		totalCommissionAmount += statement.TotalAmount
	}

	durationMs := time.Since(startTime).Milliseconds()

	slog.Info("commission calculation completed",
		"period_id", periodID,
		"reps_processed", repsProcessed,
		"total_commission_amount", totalCommissionAmount,
		"duration_ms", durationMs,
	)

	return nil
}

// GetRepDashboard aggregates a rep's current period data into the RepDashboard struct
func (s *commissionService) GetRepDashboard(ctx context.Context, repID int64) (*model.RepDashboard, error) {
	// Get current period (most recent open or closed period)
	periods, err := s.periodRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list periods: %w", err)
	}

	if len(periods) == 0 {
		return nil, fmt.Errorf("no periods found")
	}

	// Use the first (most recent) period
	period := periods[0]

	// Get goal for this rep in this period
	goal, err := s.goalRepository.GetByRepAndPeriod(ctx, repID, period.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get goal: %w", err)
	}
	if goal == nil {
		return nil, fmt.Errorf("goal not found for rep %d in period %d", repID, period.ID)
	}

	// Count acquisition and renewal events
	acquisitionActual, err := s.eventRepository.CountByRepAndType(
		ctx, repID, model.EventTypeAcquisition, period.StartDate, period.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count acquisition events: %w", err)
	}

	renewalActual, err := s.eventRepository.CountByRepAndType(
		ctx, repID, model.EventTypeRenewal, period.StartDate, period.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count renewal events: %w", err)
	}

	// Calculate attainment percentage
	attainmentPct := calculateAttainment(
		float64(acquisitionActual), float64(goal.AcquisitionTarget),
		float64(renewalActual), float64(goal.RenewalTarget),
	)

	// Calculate earned commission
	commissionEarned := calculateCommission(attainmentPct, goal.CommissionValue)

	// Get recent events
	recentEvents, err := s.eventRepository.ListByRepAndPeriod(ctx, repID, period.StartDate, period.EndDate)
	if err != nil {
		slog.Warn("failed to get recent events",
			"rep_id", repID,
			"error", err,
		)
		recentEvents = []*model.MemberEvent{}
	}

	// Get statement if it exists (for pending/approved amounts)
	commissionPending := int64(0)
	statement, err := s.statementRepository.GetByRepAndPeriod(ctx, repID, period.ID)
	if err == nil && statement != nil {
		commissionPending = statement.TotalAmount
	}

	// Ensure RecentEvents is not nil for JSON serialization
	if recentEvents == nil {
		recentEvents = []*model.MemberEvent{}
	}

	return &model.RepDashboard{
		RepID:             repID,
		PeriodName:        period.Name,
		AcquisitionGoal:   goal.AcquisitionTarget,
		AcquisitionActual: acquisitionActual,
		RenewalGoal:       goal.RenewalTarget,
		RenewalActual:     renewalActual,
		AttainmentPct:     attainmentPct,
		CommissionEarned:  commissionEarned,
		CommissionPending: commissionPending,
		RecentEvents:      recentEvents,
	}, nil
}

// GetTeamDashboard aggregates team data for a manager's direct reports
func (s *commissionService) GetTeamDashboard(ctx context.Context, managerID int64) (*model.TeamDashboard, error) {
	// Get current period
	periods, err := s.periodRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list periods: %w", err)
	}

	if len(periods) == 0 {
		return nil, fmt.Errorf("no periods found")
	}

	period := periods[0]

	// Get all users to find direct reports
	users, err := s.userRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Filter direct reports of this manager
	directReportIDs := []int64{}
	for _, user := range users {
		if user.ManagerID != nil && *user.ManagerID == managerID && user.Role == model.RoleRep {
			directReportIDs = append(directReportIDs, user.ID)
		}
	}

	// Build dashboard for each direct report
	directReports := []model.RepDashboardSummary{}
	totalCommission := int64(0)

	for _, repID := range directReportIDs {
		// Get rep info
		rep, err := s.userRepository.GetByID(ctx, repID)
		if err != nil || rep == nil {
			slog.Warn("failed to get rep", "rep_id", repID)
			continue
		}

		// Get goal
		goal, err := s.goalRepository.GetByRepAndPeriod(ctx, repID, period.ID)
		if err != nil || goal == nil {
			slog.Warn("no goal found for rep", "rep_id", repID, "period_id", period.ID)
			continue
		}

		// Count events
		acquisitionActual, err := s.eventRepository.CountByRepAndType(
			ctx, repID, model.EventTypeAcquisition, period.StartDate, period.EndDate,
		)
		if err != nil {
			slog.Warn("failed to count acquisition events", "rep_id", repID)
			acquisitionActual = 0
		}

		renewalActual, err := s.eventRepository.CountByRepAndType(
			ctx, repID, model.EventTypeRenewal, period.StartDate, period.EndDate,
		)
		if err != nil {
			slog.Warn("failed to count renewal events", "rep_id", repID)
			renewalActual = 0
		}

		// Calculate attainment
		attainmentPct := calculateAttainment(
			float64(acquisitionActual), float64(goal.AcquisitionTarget),
			float64(renewalActual), float64(goal.RenewalTarget),
		)

		// Calculate commission
		commissionEarned := calculateCommission(attainmentPct, goal.CommissionValue)

		directReports = append(directReports, model.RepDashboardSummary{
			RepID:            repID,
			RepName:          rep.Name,
			AttainmentPct:    attainmentPct,
			CommissionEarned: commissionEarned,
		})

		totalCommission += commissionEarned
	}

	return &model.TeamDashboard{
		ManagerID:       managerID,
		PeriodName:      period.Name,
		TotalCommission: totalCommission,
		DirectReports:   directReports,
	}, nil
}

// GenerateStatements creates commission statements for all reps in a period
func (s *commissionService) GenerateStatements(ctx context.Context, periodID int64) error {
	startTime := time.Now()

	// Get period
	period, err := s.periodRepository.GetByID(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to get period: %w", err)
	}
	if period == nil {
		return fmt.Errorf("period not found: %d", periodID)
	}

	// Get all goals for this period
	goals, err := s.goalRepository.ListByPeriod(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to list goals: %w", err)
	}

	statementsCreated := 0

	for _, goal := range goals {
		// Check if statement already exists
		existing, err := s.statementRepository.GetByRepAndPeriod(ctx, goal.RepID, periodID)
		if err != nil {
			slog.Error("failed to check existing statement", "rep_id", goal.RepID, "error", err)
			continue
		}

		if existing != nil {
			continue // Statement already exists
		}

		// Calculate commission
		statement, err := s.calculateRepCommission(ctx, goal, period)
		if err != nil {
			slog.Error("failed to calculate commission", "rep_id", goal.RepID, "error", err)
			continue
		}

		// Set status to pending_approval
		statement.Status = model.StatementStatusPendingApproval

		// Create statement
		if err := s.statementRepository.Create(ctx, statement); err != nil {
			slog.Error("failed to create statement", "rep_id", goal.RepID, "error", err)
			continue
		}

		statementsCreated++
	}

	durationMs := time.Since(startTime).Milliseconds()

	slog.Info("statement generation completed",
		"period_id", periodID,
		"statements_created", statementsCreated,
		"duration_ms", durationMs,
	)

	return nil
}

// ApproveStatement marks a statement as approved by finance
func (s *commissionService) ApproveStatement(ctx context.Context, statementID int64, approverID int64) error {
	return s.statementRepository.UpdateStatus(ctx, statementID, model.StatementStatusApproved, &approverID, nil)
}

// calculateRepCommission calculates commission for a single rep based on their goal and events
func (s *commissionService) calculateRepCommission(ctx context.Context, goal *model.Goal, period *model.Period) (*model.Statement, error) {
	// Count events
	acquisitionActual, err := s.eventRepository.CountByRepAndType(
		ctx, goal.RepID, model.EventTypeAcquisition, period.StartDate, period.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count acquisition events: %w", err)
	}

	renewalActual, err := s.eventRepository.CountByRepAndType(
		ctx, goal.RepID, model.EventTypeRenewal, period.StartDate, period.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count renewal events: %w", err)
	}

	// Calculate attainment percentage
	attainmentPct := calculateAttainment(
		float64(acquisitionActual), float64(goal.AcquisitionTarget),
		float64(renewalActual), float64(goal.RenewalTarget),
	)

	// Calculate commission in centavos
	totalAmount := calculateCommission(attainmentPct, goal.CommissionValue)

	return &model.Statement{
		RepID:          goal.RepID,
		PeriodID:       goal.PeriodID,
		TotalAmount:    totalAmount,
		AttainmentPct:  attainmentPct,
		Status:         model.StatementStatusDraft,
	}, nil
}

// calculateAttainment calculates the average attainment percentage for acquisition and renewal
// Formula: (acquisition_actual / acquisition_goal + renewal_actual / renewal_goal) / 2
func calculateAttainment(acqActual, acqGoal, renActual, renGoal float64) float64 {
	var acqPct float64
	if acqGoal > 0 {
		acqPct = acqActual / acqGoal
	}

	var renPct float64
	if renGoal > 0 {
		renPct = renActual / renGoal
	}

	avgPct := (acqPct + renPct) / 2.0

	// Cap attainment at 100% for calculation purposes
	if avgPct > 1.0 {
		avgPct = 1.0
	}

	return avgPct
}

// calculateCommission calculates the commission amount in centavos using proportional attainment
func calculateCommission(attainmentPct float64, commissionValue int64) int64 {
	// attainment_pct * commission_value, with rounding
	// All values in centavos (integers)
	result := int64(attainmentPct * float64(commissionValue))
	return result
}
