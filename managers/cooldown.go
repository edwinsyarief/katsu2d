package managers

import "fmt"

const (
	// DefaultCooldownLimit is the default initial capacity for a CooldownManager.
	DefaultCooldownLimit = 512

	// DEFAULT_COOLDOWN_LIMIT is kept as an alias for backward compatibility.
	DEFAULT_COOLDOWN_LIMIT = DefaultCooldownLimit
)

type cooldown struct {
	callback func()
	id       string
	duration float64
	initial  float64
}

func (c *cooldown) getRemainingRatio() float64 {
	if c.initial == 0 {
		return 0
	}
	return c.duration / c.initial
}

func (c *cooldown) getProgressRatio() float64 {
	if c.initial == 0 {
		return 0
	}
	return 1 - c.getRemainingRatio()
}

// CooldownManager tracks named cooldown timers and invokes callbacks upon expiry.
type CooldownManager struct {
	cds        []*cooldown
	pendingCbs []func()
	maxSize    int
}

// NewCooldownManager creates a CooldownManager with an initial capacity hint of maxSize.
func NewCooldownManager(maxSize int) *CooldownManager {
	if maxSize <= 0 {
		maxSize = DefaultCooldownLimit
	}
	return &CooldownManager{
		cds:     make([]*cooldown, 0, maxSize),
		maxSize: maxSize,
	}
}

// Has reports whether a cooldown with the given id is currently active.
func (m *CooldownManager) Has(id string) bool {
	return m.get(id) != nil
}

// GetProgress returns the normalized completion progress [0, 1] for the named cooldown.
func (m *CooldownManager) GetProgress(id string) float64 {
	c := m.get(id)
	if c == nil {
		panic(fmt.Sprintf("Cooldown %v doesn't exist!", id))
	}
	if c.duration <= 0 {
		return 1
	}
	return c.getProgressRatio()
}

// GetRatio returns the normalized remaining time ratio [0, 1] for the named cooldown.
func (m *CooldownManager) GetRatio(id string) float64 {
	c := m.get(id)
	if c == nil {
		panic(fmt.Sprintf("Cooldown %v doesn't exist!", id))
	}
	if c.duration <= 0 {
		return 0
	}
	return c.getRemainingRatio()
}

// Set registers a new cooldown if duration > 0 and id is not already active.
func (m *CooldownManager) Set(id string, duration float64, callback func()) {
	if duration <= 0 {
		return
	}
	if m.Has(id) {
		return
	}
	m.cds = append(m.cds, &cooldown{
		id:       id,
		duration: duration,
		initial:  duration,
		callback: callback,
	})
}

// Override resets the duration of an existing cooldown to duration.
func (m *CooldownManager) Override(id string, duration float64) {
	if duration <= 0 {
		return
	}
	c := m.get(id)
	if c == nil {
		panic(fmt.Sprintf("Cooldown %v doesn't exist!", id))
	}
	c.duration = duration
	c.initial = duration
}

// Update advances all active cooldowns by delta, removes expired entries, and invokes their callbacks.
func (m *CooldownManager) Update(delta float64) {
	m.pendingCbs = m.pendingCbs[:0]
	n := 0
	for _, c := range m.cds {
		if c == nil {
			continue
		}
		c.duration -= delta
		if c.duration <= 0 {
			if c.callback != nil {
				m.pendingCbs = append(m.pendingCbs, c.callback)
			}
			continue
		}
		m.cds[n] = c
		n++
	}
	clear(m.cds[n:])
	m.cds = m.cds[:n]

	for i, cb := range m.pendingCbs {
		if cb != nil {
			cb()
			m.pendingCbs[i] = nil
		}
	}
	m.pendingCbs = m.pendingCbs[:0]
}

// Reset clears all active cooldowns while retaining slice capacity.
func (m *CooldownManager) Reset() {
	clear(m.cds)
	m.cds = m.cds[:0]
	clear(m.pendingCbs)
	m.pendingCbs = m.pendingCbs[:0]
}

func (m *CooldownManager) get(id string) *cooldown {
	for _, c := range m.cds {
		if c != nil && c.id == id {
			return c
		}
	}
	return nil
}
