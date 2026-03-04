package resource

// bestEffort logs a warning if a best-effort operation fails.
// Does nothing if err is nil.
func (s *Service) bestEffort(op string, err error) {
	if err == nil {
		return
	}
	s.logger.Warn("best-effort operation failed", "op", op, "error", err)
}

// rollback collects best-effort undo operations to run on failure.
type rollback struct {
	svc     *Service
	actions []func()
	active  bool
}

// newRollback creates a new active rollback tied to the given service.
func newRollback(svc *Service) *rollback {
	return &rollback{svc: svc, active: true}
}

// add registers an undo operation with a label.
func (rb *rollback) add(label string, fn func() error) {
	rb.actions = append(rb.actions, func() {
		rb.svc.bestEffort(label+" rollback", fn())
	})
}

// execute runs all registered undo operations in reverse order, if still armed.
func (rb *rollback) execute() {
	if !rb.active {
		return
	}
	for i := len(rb.actions) - 1; i >= 0; i-- {
		rb.actions[i]()
	}
}

// disarm prevents the rollback from executing (call on success).
func (rb *rollback) disarm() {
	rb.active = false
}
