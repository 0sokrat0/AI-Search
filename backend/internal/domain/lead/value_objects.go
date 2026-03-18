package lead

type Status string

const (
	StatusNew           Status = "new"
	StatusDetected      Status = "detected"
	StatusConfirmed     Status = "confirmed"
	StatusControversial Status = "controversial"
	StatusFalsePositive Status = "false_positive"
	StatusContacted     Status = "contacted"
	StatusQualified     Status = "qualified"
	StatusConverted     Status = "converted"
	StatusRejected      Status = "rejected"
)

type QualificationSource string

const (
	QualificationSourceNone   QualificationSource = ""
	QualificationSourceAI     QualificationSource = "ai_qualified"
	QualificationSourceManual QualificationSource = "manual_approved"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

func PriorityFromScore(score float64) Priority {
	switch {
	case score >= 0.85:
		return PriorityUrgent
	case score >= 0.70:
		return PriorityHigh
	case score >= 0.50:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

func validateTransition(from, to Status) error {
	if !IsValidStatus(to) {
		return ErrInvalidStatus
	}

	if from == StatusConverted || from == StatusRejected {
		return ErrInvalidStatusTransition
	}
	return nil
}

func IsValidStatus(s Status) bool {
	switch s {
	case StatusNew, StatusDetected, StatusConfirmed, StatusControversial, StatusFalsePositive,
		StatusContacted, StatusQualified, StatusConverted, StatusRejected:
		return true
	default:
		return false
	}
}

func ensureSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func copySlice(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
