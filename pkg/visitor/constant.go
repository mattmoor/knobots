package visitor

// VisitControl is an enum that determines whether visitation should
// continue or cease after each callback.
type VisitControl string

const (
	Continue VisitControl = "continue"
	Break    VisitControl = "break"
)
